package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	
	"bakashier/data"
	"bakashier/utils"
	"bakashier/view"
)


// キューからメッセージを受け取り、ワーカーにジョブを配分する。
// FIND_DIR でジョブを投入し、全ジョブが FINISH_JOB で完了すると各ワーカーに EXIT を送る。
func backupManager(workers uint32, fromWorkerQueue <-chan messageFromWorkerToManager, toWorkerQueue chan messageFromManagerToWorker, toViewQueue chan<- view.MessageToView, fromViewQueue <-chan view.MessageToManager, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		toViewQueue <- view.MessageToView{
			Source:   view.MANAGER,
			MsgType:  view.FINISHED,
			WorkerId: 0,
			Detail:   "",
		}
	}()
	
	var untreated int = 0
	var untreatedMessage = []messageFromManagerToWorker{}
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
				untreatedMessage = append(untreatedMessage, messageFromManagerToWorker{
					MsgType: NEXT_JOB,
					SrcDir:  msg.SrcDir,
					DistDir: msg.DistDir,
					Detail:  msg.Detail,
				})
				untreated++
			case FINISH_JOB:
				untreated--
			case ERROR:
				toViewQueue <- view.MessageToView{
					Source:   view.MANAGER,
					MsgType:  view.ERROR,
					WorkerId: msg.WorkerId,
					Detail:   msg.Detail,
				}
			}
		default:
		}
		
		// 一時停止中の場合は、ワーカーに送ったメッセージをすべて未処理に移動
		if stopWorkers {
			func() {
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
			func() {
				for {
					select {
					case msg := <-toWorkerQueue:
						if msg.MsgType != EXIT {
							untreated--
						}
					default:
						return
					}
				}
			}()
			messages := len(untreatedMessage)
			untreatedMessage = make([]messageFromManagerToWorker, 0)
			untreated -= messages
			
			for i := 0; i < untreated; i++ {
				toWorkerQueue <- messageFromManagerToWorker{
					MsgType: EXIT,
					SrcDir:  "",
					DistDir: "",
					Detail:  "",
				}
			}
		}
		
		// 全ジョブが完了した場合は、各ワーカーに EXIT を送って終了
		if untreated <= 0 {
			for i := uint32(0); i < workers; i++ {
				toWorkerQueue <- messageFromManagerToWorker{
					MsgType: EXIT,
					SrcDir:  "",
					DistDir: "",
					Detail:  "",
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
func backupWorker(workerId uint, password string, toManagerQueue chan<- messageFromWorkerToManager, fromManagerQueue <-chan messageFromManagerToWorker, toViewQueue chan<- view.MessageToView, wg *sync.WaitGroup, chunkSize uint64, limit SettingsLimit) {
	defer wg.Done()
	var processedSize uint64 = 0
	
	toViewQueue <- view.MessageToView{
		Source:   view.WORKER,
		MsgType:  view.ADD_WORKER,
		WorkerId: workerId,
		Detail:   "",
	}
	
	for {
		queue := <-fromManagerQueue
		if queue.MsgType == EXIT {
			break
		}
		
		var errHandler = func(prefix string, err error) {
			toManagerQueue <- messageFromWorkerToManager{
				WorkerId: workerId,
				MsgType:  ERROR,
				SrcDir:   queue.SrcDir,
				DistDir:  queue.DistDir,
				Detail:   fmt.Sprintf("%s: %s", prefix, err.Error()),
			}
		}
		
		toViewQueue <- view.MessageToView{
			Source:   view.WORKER,
			MsgType:  view.START_DIR,
			WorkerId: workerId,
			SrcPath:  queue.SrcDir,
			DistPath: queue.DistDir,
			Detail:   "",
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
			
			nameMap := make(map[string]string)                 // [HideName]RealName
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
				entry := data.DirectoryEntry{Type: data.Unknown}
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
						Type:     data.Directory,
						RealName: file.Name(),
						HideName: hideName,
						Size:     uint64(0),
						ModTime:  time.Now(),
					}
					
					// 既存のエントリと異なる場合は変更があると判定 または バックアップ先にディレクトリが存在しない場合は変更があると判定
					if entry.Type != data.Directory || entry.RealName != file.Name() {
						isExistChanges = true
					} else if _, err := os.Stat(filepath.Join(queue.DistDir, hideName)); err != nil {
						isExistChanges = true
					}
					
					// 子ディレクトリの発見をディスパッチャに通知
					toManagerQueue <- messageFromWorkerToManager{
						WorkerId: workerId,
						MsgType:  FIND_DIR,
						SrcDir:   filepath.Join(queue.SrcDir, file.Name()),
						DistDir:  filepath.Join(queue.DistDir, hideName),
						Detail:   "",
					}
				} else {
					// ファイル処理開始をビューに通知
					toViewQueue <- view.MessageToView{
						Source:   view.WORKER,
						MsgType:  view.START_FILE,
						WorkerId: workerId,
						SrcPath:  filepath.Join(queue.SrcDir, file.Name()),
						DistPath: filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", hideName)),
						Detail:   "",
					}
					
					func() {
						isNotChangeFile := false
						fileInfo, err := file.Info()
						if err != nil {
							errHandler("Failed to get file info", err)
							return
						}
						
						// 変更がないか
						if entry.Type == data.File {
							if entry.Size == uint64(fileInfo.Size()) && entry.ModTime.Equal(fileInfo.ModTime()) {
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
						isExistChanges = true
						
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
							Type:     data.File,
							RealName: file.Name(),
							HideName: hideName,
							Size:     uint64(fileInfo.Size()),
							ModTime:  fileInfo.ModTime(),
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
						Source:   view.WORKER,
						MsgType:  view.FINISH_FILE,
						WorkerId: workerId,
						SrcPath:  filepath.Join(queue.SrcDir, file.Name()),
						DistPath: filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", hideName)),
						Detail:   "",
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
			Source:   view.WORKER,
			MsgType:  view.FINISH_DIR,
			WorkerId: workerId,
			SrcPath:  queue.SrcDir,
			DistPath: queue.DistDir,
			Detail:   "",
		}
		toManagerQueue <- messageFromWorkerToManager{
			WorkerId: workerId,
			MsgType:  FINISH_JOB,
			SrcDir:   queue.SrcDir,
			DistDir:  queue.DistDir,
			Detail:   "",
		}
	}
}

// settings.SrcDir を暗号化・圧縮して settings.DistDir にバックアップする。
// マネージャ1つと複数のワーカーを起動し、チャネルでジョブを分配する。
func Backup(settings Settings, toViewQueue chan<- view.MessageToView, fromViewQueue <-chan view.MessageToManager) {
	var wg sync.WaitGroup
	
	workers := settings.Workers
	queueSize := workers * 8
	if workers <= 0 {
		workers = 1
		queueSize = 8
	}
	
	workerToManagerQueue := make(chan messageFromWorkerToManager, queueSize)
	managerToWorkerQueue := make(chan messageFromManagerToWorker, queueSize)
	
	workerToManagerQueue <- messageFromWorkerToManager{
		MsgType: FIND_DIR,
		SrcDir:  settings.SrcDir,
		DistDir: settings.DistDir,
		Detail:  "",
	}
	
	wg.Add(int(workers) + 1)
	go backupManager(workers, workerToManagerQueue, managerToWorkerQueue, toViewQueue, fromViewQueue, &wg)
	for i := uint(0); i < uint(workers); i++ {
		go backupWorker(i+1, settings.Password, workerToManagerQueue, managerToWorkerQueue, toViewQueue, &wg, settings.ChunkSize, settings.Limit)
	}
	wg.Wait()
	
	close(workerToManagerQueue)
	close(managerToWorkerQueue)
}
