package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"bakashier/data"
)

// ディスパッチャキューからメッセージを受け取り、ワーカーにジョブを配分する。
// 全ジョブ完了後に各ワーカーに EXIT を送って終了する。
func restoreDispatcher(workers int, dispatcherQueue <-chan dispatcherMessage, workerQueue chan<- workerMessage, wg *sync.WaitGroup) {
	defer wg.Done()

	var untreated int = 0
	var untreatedMessage = []workerMessage{}
	for {
		msg := <-dispatcherQueue

		switch msg.MsgType {
		case FIND_DIR:
			untreatedMessage = append(untreatedMessage, workerMessage{
				MsgType: NEXT_JOB,
				SrcDir:  msg.SrcDir,
				DistDir: msg.DistDir,
				Detail:  msg.Detail,
			})
			untreated++
		case FINISH_JOB:
			untreated--
		case ERROR:
			fmt.Println(msg.Detail)
		}

		if untreated <= 0 {
			for i := 0; i < workers; i++ {
				workerQueue <- workerMessage{
					MsgType: EXIT,
					SrcDir:  "",
					DistDir: "",
					Detail:  "",
				}
			}
			break
		}
		
		for {
			if len(dispatcherQueue) != 0 { break }
			if len(untreatedMessage) == 0 { break }
			
			msg := untreatedMessage[0]
			select {
				case workerQueue <- msg:
					untreatedMessage = untreatedMessage[1:]
				default:
					time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

// ワーカーキューからジョブを受け取り、_directory_.bks と .bks ファイルから復元する。
// ディレクトリエントリに従い、隠し名の .bks を復号して実名で distDir に書き出す。
func restoreWorker(password string, dispatcherQueue chan<- dispatcherMessage, workerQueue <-chan workerMessage, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		queue := <-workerQueue
		if queue.MsgType == EXIT {
			break
		}
		
		var errHandler = func(prefix string, err error) {
			dispatcherQueue <- dispatcherMessage{
				MsgType: ERROR,
				SrcDir:  queue.SrcDir,
				DistDir: queue.DistDir,
				Detail:  fmt.Sprintf("%s: %s", prefix, err.Error()),
			}
		}
		
		func() {
			err := os.MkdirAll(queue.DistDir, 0755)
			if err != nil {
				errHandler("Failed to create directory", err)
				return
			}
			
			directoryEntryFile := filepath.Join(queue.SrcDir, "_directory_.bks")
			entries, err := loadDirectoryEntries(directoryEntryFile, password)
			if err != nil {
				errHandler("Failed to load directory entries", err)
				return
			}
			
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
					
					dispatcherQueue <- dispatcherMessage{
						MsgType: FIND_DIR,
						SrcDir:  hiddenDir,
						DistDir: realDir,
						Detail:  "",
					}
					fmt.Printf("Successfully restored directory: %s -> %s\n", hiddenDir, realDir)
				case data.File:
					archiveFile := filepath.Join(queue.SrcDir, fmt.Sprintf("%s.bks", entry.HideName))
					
					err, realFile := data.ImportStreamArchive(archiveFile, queue.DistDir, password)
					if err != nil {
						errHandler("Failed to import stream archive", err)
						return
					}
					_ = os.Chtimes(realFile, time.Now(), entry.ModTime)
					
					fmt.Printf("Successfully restored: %s -> %s\n", filepath.Join(queue.SrcDir, fmt.Sprintf("%s.bks", entry.HideName)), realFile)
				default:
					errHandler("Unknown entry type", fmt.Errorf("%v", entry.Type))
					return
				}
			}
		}()
		
		dispatcherQueue <- dispatcherMessage{
			MsgType: FINISH_JOB,
			SrcDir:  queue.SrcDir,
			DistDir: queue.DistDir,
			Detail:  "",
		}
	}
}
