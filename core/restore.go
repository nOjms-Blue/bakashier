package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	
	"bakashier/data"
	"bakashier/view"
)


// ディスパッチャキューからメッセージを受け取り、ワーカーにジョブを配分する。
// 全ジョブ完了後に各ワーカーに EXIT を送って終了する。
func restoreDispatcher(workers int, fromWorkerQueue <-chan messageFromWorkerToDispatcher, toWorkerQueue chan messageFromDispatcherToWorker, toViewQueue chan<- view.MessageToView, fromViewQueue <-chan view.MessageToDispatcher, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		toViewQueue <- view.MessageToView{
			Source:   view.DISPATCHER,
			MsgType:  view.FINISHED,
			WorkerId: 0,
			Detail:   "",
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
					SrcDir:  msg.SrcDir,
					DistDir: msg.DistDir,
					Detail:  msg.Detail,
				})
				untreated++
			case FINISH_JOB:
				untreated--
			case ERROR:
				toViewQueue <- view.MessageToView{
					Source:   view.DISPATCHER,
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
			untreatedMessage = make([]messageFromDispatcherToWorker, 0)
			untreated -= messages
			
			for i := 0; i < untreated; i++ {
				toWorkerQueue <- messageFromDispatcherToWorker{
					MsgType: EXIT,
					SrcDir:  "",
					DistDir: "",
					Detail:  "",
				}
			}
		}
		
		// 全ジョブが完了した場合は、各ワーカーに EXIT を送って終了
		if untreated <= 0 {
			for i := 0; i < workers; i++ {
				toWorkerQueue <- messageFromDispatcherToWorker{
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
			if len(fromViewQueue) != 0 {
				break
			}
			if len(fromWorkerQueue) != 0 {
				break
			}
			if len(untreatedMessage) == 0 {
				break
			}
			
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

// ワーカーキューからジョブを受け取り、_directory_.bks と .bks ファイルから復元する。
// ディレクトリエントリに従い、隠し名の .bks を復号して実名で distDir に書き出す。
func restoreWorker(workerId uint, password string, toDispatcherQueue chan<- messageFromWorkerToDispatcher, fromDispatcherQueue <-chan messageFromDispatcherToWorker, toViewQueue chan<- view.MessageToView, wg *sync.WaitGroup, limit Limit) {
	defer wg.Done()
	var processedSize uint64 = 0
	
	toViewQueue <- view.MessageToView{
		Source:   view.WORKER,
		MsgType:  view.ADD_WORKER,
		WorkerId: workerId,
		Detail:   "",
	}
	
	for {
		queue := <-fromDispatcherQueue
		if queue.MsgType == EXIT { break }
		
		var errHandler = func(prefix string, err error) {
			toDispatcherQueue <- messageFromWorkerToDispatcher{
				WorkerId: workerId,
				MsgType:  ERROR,
				SrcDir:   queue.SrcDir,
				DistDir:  queue.DistDir,
				Detail:   fmt.Sprintf("%s: %s", prefix, err.Error()),
			}
		}
		
		// ディレクトリ処理開始をビューに通知
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
			
			// _directory_.bks からエントリ一覧を読み込む。
			directoryEntryFile := filepath.Join(queue.SrcDir, "_directory_.bks")
			entries, err := loadDirectoryEntries(directoryEntryFile, password)
			if err != nil {
				errHandler("Failed to load directory entries", err)
				return
			}
			
			// リストアを実行
			for _, entry := range entries {
				switch entry.Type {
				case data.Directory:
					hiddenDir := filepath.Join(queue.SrcDir, entry.HideName)
					realDir := filepath.Join(queue.DistDir, entry.RealName)
					err = os.MkdirAll(realDir, 0755)
					if err != nil {
						errHandler("Failed to create directory", err)
						return
					}
					
					// 子ディレクトリの発見をディスパッチャに通知
					toDispatcherQueue <- messageFromWorkerToDispatcher{
						WorkerId: workerId,
						MsgType:  FIND_DIR,
						SrcDir:   hiddenDir,
						DistDir:  realDir,
						Detail:   "",
					}
				case data.File:
					archiveFile := filepath.Join(queue.SrcDir, fmt.Sprintf("%s.bks", entry.HideName))
					
					// ファイル処理開始をビューに通知
					toViewQueue <- view.MessageToView{
						Source: view.WORKER,
						MsgType: view.START_FILE,
						WorkerId: workerId,
						SrcPath: archiveFile,
						DistPath: filepath.Join(queue.DistDir, entry.RealName),
						Detail: "",
					}
					
					func () {
						err, realFile := data.ImportStreamArchive(archiveFile, queue.DistDir, password)
						if err != nil {
							errHandler("Failed to import stream archive", err)
							return
						}
						_ = os.Chtimes(realFile, time.Now(), entry.ModTime)
						
						if limit.Size > 0 && limit.Wait > 0 {
							processedSize += uint64(entry.Size)
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
						SrcPath: archiveFile,
						DistPath: filepath.Join(queue.DistDir, entry.RealName),
						Detail: "",
					}
				default:
					errHandler("Unknown entry type", fmt.Errorf("%v", entry.Type))
					return
				}
			}
		}()
		
		// ディレクトリ処理完了をビューに通知
		toViewQueue <- view.MessageToView{
			Source:   view.WORKER,
			MsgType:  view.FINISH_DIR,
			WorkerId: workerId,
			SrcPath:  queue.SrcDir,
			DistPath: queue.DistDir,
			Detail:   "",
		}
		
		// ディレクトリ処理完了をディスパッチャに通知
		toDispatcherQueue <- messageFromWorkerToDispatcher{
			WorkerId: workerId,
			MsgType:  FINISH_JOB,
			SrcDir:   queue.SrcDir,
			DistDir:  queue.DistDir,
			Detail:   "",
		}
	}
}
