package core

import (
	"fmt"
	"runtime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	
	"bakashier/data"
	"bakashier/utils"
	"bakashier/view"
)


// ディスパッチャキューからメッセージを受け取り、ワーカーにジョブを配分する。
// FIND_DIR でジョブを投入し、全ジョブが FINISH_JOB で完了すると各ワーカーに EXIT を送る。
func backupDispatcher(workers int, fromWorkerQueue <-chan messageFromWorkerToDispatcher, toWorkerQueue chan messageFromDispatcherToWorker, toViewQueue chan<- view.MessageToView, fromViewQueue <-chan view.MessageToDispatcher, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		toViewQueue <- view.MessageToView{
			Source: view.DISPATCHER,
			MsgType: view.FINISHED,
			WorkerId: 0,
			Detail: "",
		}
	}()
	
	var untreated int = 0
	var untreatedMessage = []messageFromDispatcherToWorker{}
	var stopWorkers bool = false
	var termination bool = false
	for {
		select {
		case msg := <-fromViewQueue:
			// ビューからのメッセージを処理する
			switch msg.MsgType {
			case view.STOP_WORKERS:
				stopWorkers = true
			case view.RESUME_WORKERS:
				stopWorkers = false
			case view.TERMINATION:
				termination = true
			}
		case msg := <-fromWorkerQueue:
			// ワーカーからのメッセージを処理する
			switch msg.MsgType {
			case FIND_DIR:
				untreatedMessage = append(untreatedMessage, messageFromDispatcherToWorker{
					MsgType: NEXT_JOB,
					SrcDir: msg.SrcDir,
					DistDir: msg.DistDir,
					Detail: msg.Detail,
				})
				untreated++
			case FINISH_JOB:
				untreated--
			case ERROR:
				toViewQueue <- view.MessageToView{
					Source: view.DISPATCHER,
					MsgType: view.ERROR,
					WorkerId: msg.WorkerId,
					Detail: msg.Detail,
				}
			}
		default:
		}
		
		// 一時停止中の場合は、ワーカーに送ったメッセージをすべて未処理に移動
		if stopWorkers {
			func () {
				for {
					select {
					case msg := <-toWorkerQueue:
						untreatedMessage = append(untreatedMessage, msg)
					default:
						return
					}
				}
			}()
		}
		
		// 終了指示が来た場合は、ワーカーに送ったメッセージをすべて破棄して終了指示を送る
		if termination {
			// ワーカーに送ったメッセージをすべて破棄して終了指示を送る
			func () {
				for {
					select {
					case msg := <-toWorkerQueue:
						if msg.MsgType != EXIT { untreated-- }
					default:
						return
					}
				}
			}()
			messages := len(untreatedMessage)
			untreatedMessage = make([]messageFromDispatcherToWorker, 0)
			untreated -= messages
			
			for i := 0; i < untreated; i++ {
				toWorkerQueue <- messageFromDispatcherToWorker{
					MsgType: EXIT,
					SrcDir: "",
					DistDir: "",
					Detail: "",
				}
			}
		}
		
		// 全ジョブが完了した場合は、各ワーカーに EXIT を送って終了
		if untreated <= 0 {
			for i := 0; i < workers; i++ {
				toWorkerQueue <- messageFromDispatcherToWorker{
					MsgType: EXIT,
					SrcDir: "",
					DistDir: "",
					Detail: "",
				}
			}
			break
		}
		
		// ジョブの配分
		for {
			// メッセージが来た場合は、受信メッセージを先に処理する
			if len(fromViewQueue) != 0 { break }
			if len(fromWorkerQueue) != 0 { break }
			if len(untreatedMessage) == 0 { break }
			
			// 一時停止中または終了指示が来た場合は、ジョブを配分しない
			if stopWorkers || termination {
				time.Sleep(10 * time.Millisecond)
				break
			}
			
			// 未処理のジョブを配分する
			msg := untreatedMessage[0]
			select {
				case toWorkerQueue <- msg:
					untreatedMessage = untreatedMessage[1:]
				default:
					time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

// ワーカーキューからジョブを受け取り、ディレクトリを走査してファイルをアーカイブする。
// 既存の _directory_.bks を読み、変更のないファイルはスキップする。ディレクトリは FIND_DIR で再投入する。
func backupWorker(workerId uint, password string, toDispatcherQueue chan<- messageFromWorkerToDispatcher, fromDispatcherQueue <-chan messageFromDispatcherToWorker, toViewQueue chan<- view.MessageToView, wg *sync.WaitGroup, chunkSize uint64, limit Limit) {
	defer wg.Done()
	var processedSize uint64 = 0
	
	toViewQueue <- view.MessageToView{
		Source: view.WORKER,
		MsgType: view.ADD_WORKER,
		WorkerId: workerId,
		Detail: "",
	}
	
	for {
		queue := <-fromDispatcherQueue
		if queue.MsgType == EXIT { break }
		
		var errHandler = func(prefix string, err error) {
			toDispatcherQueue <- messageFromWorkerToDispatcher{
				WorkerId: workerId,
				MsgType: ERROR,
				SrcDir: queue.SrcDir,
				DistDir: queue.DistDir,
				Detail: fmt.Sprintf("%s: %s", prefix, err.Error()),
			}
		}
		
		toViewQueue <- view.MessageToView{
			Source: view.WORKER,
			MsgType: view.START_DIR,
			WorkerId: workerId,
			SrcPath: queue.SrcDir,
			DistPath: queue.DistDir,
			Detail: "",
		}
		
		func() {
			err := os.MkdirAll(queue.DistDir, 0755)
			if err != nil {
				errHandler("Failed to create directory", err)
				return
			}
			
			files, err := os.ReadDir(queue.SrcDir)
			if err != nil {
				errHandler("Failed to read directory", err)
				return
			}
			
			nameMap := make(map[string]string) // [HideName]RealName
			newEntries := make(map[string]data.DirectoryEntry) // [HideName]DirectoryEntry
			directoryEntryFile := filepath.Join(queue.DistDir, "_directory_.bks")
			
			// 既存の _directory_.bks が存在しない場合は、中断されたバックアップを削除する。
			if _, err := os.Stat(directoryEntryFile); err != nil {
				items, err := os.ReadDir(queue.DistDir)
				if err == nil {
					for _, item := range items {
						if item.IsDir() {
							os.RemoveAll(filepath.Join(queue.DistDir, item.Name()))
						} else {
							os.Remove(filepath.Join(queue.DistDir, item.Name()))
						}
					}
				}
			}
			
			// 既存の _directory_.bks からエントリ一覧を読み込む。
			entries, err := loadDirectoryEntries(directoryEntryFile, password)
			if err != nil {
				errHandler("Failed to load directory entries", err)
				return
			}
			isExistEntries := len(entries) > 0
			for _, entry := range entries {
				nameMap[entry.HideName] = entry.RealName
			}
			
			// バックアップの実行
			isExistChanges := false
			for _, file := range files {
				hideName := utils.GenerateUniqueRandomName(nameMap)
				entry := data.DirectoryEntry{ Type: data.Unknown }
				for _, registered := range entries {
					if registered.RealName == file.Name() {
						hideName = registered.HideName
						entry = registered
						break
					}
				}
				nameMap[hideName] = file.Name()
				
				if file.IsDir() {
					// ディレクトリエントリを追加
					newEntries[hideName] = data.DirectoryEntry{
						Type: data.Directory,
						RealName: file.Name(),
						HideName: hideName,
						Size: uint64(0),
						ModTime: time.Now(),
					}
					
					// 既存のエントリと異なる場合は変更があると判定 または バックアップ先にディレクトリが存在しない場合は変更があると判定
					if entry.Type != data.Directory || entry.RealName != file.Name() {
						isExistChanges = true
					} else if _, err := os.Stat(filepath.Join(queue.DistDir, hideName)); err != nil {
						isExistChanges = true
					}
					
					// 子ディレクトリの発見をディスパッチャに通知
					toDispatcherQueue <- messageFromWorkerToDispatcher{
						WorkerId: workerId,
						MsgType: FIND_DIR,
						SrcDir: filepath.Join(queue.SrcDir, file.Name()),
						DistDir: filepath.Join(queue.DistDir, hideName),
						Detail: "",
					}
				} else {
					// ファイル処理開始をビューに通知
					toViewQueue <- view.MessageToView{
						Source: view.WORKER,
						MsgType: view.START_FILE,
						WorkerId: workerId,
						SrcPath: filepath.Join(queue.SrcDir, file.Name()),
						DistPath: filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", hideName)),
						Detail: "",
					}
					
					func() {
						isNotChangeFile := false
						fileInfo, err := file.Info()
						if err != nil {
							errHandler("Failed to get file info", err)
							return
						}
						
						// 変更がないか
						if (entry.Type == data.File) {
							if (entry.Size == uint64(fileInfo.Size()) && entry.ModTime.Equal(fileInfo.ModTime())) {
								isNotChangeFile = true
							}
						}
						
						// バックアップ先にファイルが存在しない場合は変更があると判定
						if isNotChangeFile {
							if _, err := os.Stat(filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", hideName))); err != nil {
								isNotChangeFile = false
							}
						}
						
						// 変更がない場合はスキップ
						if isNotChangeFile {
							newEntries[hideName] = entry
							return
						}
						
						// ファイルをバックアップ
						srcFile := filepath.Join(queue.SrcDir, file.Name())
						archiveFile := filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", hideName))
						err = data.ExportStreamArchive(srcFile, archiveFile, file.Name(), password, chunkSize)
						if err != nil {
							errHandler("Failed to export stream archive", err)
							return
						}
						
						// ファイルエントリを追加
						newEntries[hideName] = data.DirectoryEntry{
							Type: data.File,
							RealName: file.Name(),
							HideName: hideName,
							Size: uint64(fileInfo.Size()),
							ModTime: fileInfo.ModTime(),
						}
						
						if limit.Size > 0 && limit.Wait > 0 {
							processedSize += uint64(fileInfo.Size())
							if limit.Size > 0 && processedSize >= limit.Size {
								time.Sleep(time.Duration(limit.Wait) * time.Second)
								processedSize = processedSize - limit.Size
							}
						}
					}()
					
					// ファイル処理完了をビューに通知
					toViewQueue <- view.MessageToView{
						Source: view.WORKER,
						MsgType: view.FINISH_FILE,
						WorkerId: workerId,
						SrcPath: filepath.Join(queue.SrcDir, file.Name()),
						DistPath: filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", hideName)),
						Detail: "",
					}
				}
			}
			
			// 既存のエントリから削除されたファイルを削除する。
			if isExistEntries {
				for _, entry := range entries {
					if _, ok := newEntries[entry.HideName]; !ok {
						isExistChanges = true
						if entry.Type == data.File {
							os.Remove(filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", entry.HideName)))
						} else {
							os.RemoveAll(filepath.Join(queue.DistDir, entry.HideName))
						}
					}
				}
			}
			
			// エントリに存在しないバックアップファイルを削除
			dstFiles, err := os.ReadDir(queue.DistDir)
			if err != nil {
				errHandler("Failed to read backup directory", err)
				return
			}
			for _, dstFile := range dstFiles {
				isExist := false
				// ファイル名の先頭と末尾が_の場合はスキップ
				if !dstFile.IsDir() {
					name := strings.ToLower(dstFile.Name())
					if strings.HasPrefix(name, "_") && strings.HasSuffix(name, "_.bks") {
						continue
					}
				}
				
				for _, entry := range newEntries {
					if entry.Type == data.File {
						if fmt.Sprintf("%s.bks", entry.HideName) == dstFile.Name() {
							isExist = true
							break
						}
					} else {
						if entry.HideName == dstFile.Name() {
							isExist = true
							break
						}
					}
				}
				
				if !isExist {
					if dstFile.IsDir() {
						os.RemoveAll(filepath.Join(queue.DistDir, dstFile.Name()))
					} else {
						os.Remove(filepath.Join(queue.DistDir, dstFile.Name()))
					}
				}
			}
			
			// ディレクトリエントリを保存
			if isExistChanges {
				entries = make([]data.DirectoryEntry, 0, len(newEntries))
				for _, entry := range newEntries {
					entries = append(entries, entry)
				}
				content, err := data.ExportDirectoryEntries(entries)
				if err != nil {
					errHandler("Failed to export directory entries", err)
					return
				}
				archive, err := data.ToArchiveData(queue.SrcDir, content, password)
				if err != nil {
					errHandler("Failed to create export directory entries archive data", err)
					return
				}
				err = archive.Export(directoryEntryFile)
				if err != nil {
					errHandler("Failed to export directory entries archive", err)
					return
				}
			}
		}()
		
		toViewQueue <- view.MessageToView{
			Source: view.WORKER,
			MsgType: view.FINISH_DIR,
			WorkerId: workerId,
			SrcPath: queue.SrcDir,
			DistPath: queue.DistDir,
			Detail: "",
		}
		toDispatcherQueue <- messageFromWorkerToDispatcher{
			WorkerId: workerId,
			MsgType: FINISH_JOB,
			SrcDir: queue.SrcDir,
			DistDir: queue.DistDir,
			Detail: "",
		}
	}
}

// srcDir を暗号化・圧縮して distDir にバックアップする。
// ディスパッチャ1つとワーカー4つを起動し、チャネルでジョブを分配する。
func Backup(srcDir string, distDir string, password string, chunkSize uint64, limit Limit, toViewQueue chan<- view.MessageToView, fromViewQueue <-chan view.MessageToDispatcher) {
	var wg sync.WaitGroup
	var workers int = runtime.GOMAXPROCS(0)
	var queueSize int = workers * 8
	
	if workers <= 0 {
		workers = 1
		queueSize = 8
	}
	
	workerToDispatcherQueue := make(chan messageFromWorkerToDispatcher, queueSize)
	dispatcherToWorkerQueue := make(chan messageFromDispatcherToWorker, queueSize)
	
	workerToDispatcherQueue <- messageFromWorkerToDispatcher{
		MsgType: FIND_DIR,
		SrcDir: srcDir,
		DistDir: distDir,
		Detail: "",
	}
	
	wg.Add(workers + 1)
	go backupDispatcher(workers, workerToDispatcherQueue, dispatcherToWorkerQueue, toViewQueue, fromViewQueue, &wg)
	for i := uint(0); i < uint(workers); i++ {
		go backupWorker(i + 1, password, workerToDispatcherQueue, dispatcherToWorkerQueue, toViewQueue, &wg, chunkSize, limit)
	}
	wg.Wait()
	
	close(workerToDispatcherQueue)
	close(dispatcherToWorkerQueue)
}