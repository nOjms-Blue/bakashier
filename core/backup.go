package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	
	"bakashier/data"
	"bakashier/utils"
)

func backupDispatcher(workers int, dispatcherQueue <-chan dispatcherMessage, workerQueue chan<- workerMessage, wg *sync.WaitGroup) {
	defer wg.Done()
	
	var untreated int = 0
	for ;; {
		msg := <-dispatcherQueue
		
		switch msg.MsgType {
		case FIND_DIR:
			workerQueue <- workerMessage{
				MsgType: NEXT_JOB,
				SrcDir: msg.SrcDir,
				DistDir: msg.DistDir,
				Detail: msg.Detail,
			}
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
					SrcDir: "",
					DistDir: "",
					Detail: "",
				}
			}
			break
		}
	}
}

func backupWorker(password string, dispatcherQueue chan<- dispatcherMessage, workerQueue <-chan workerMessage, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for ;; {
		queue := <-workerQueue
		if (queue.MsgType == EXIT) { break }
		
		var errHandler = func(prefix string, err error) {
			dispatcherQueue <- dispatcherMessage{
				MsgType: ERROR,
				SrcDir: queue.SrcDir,
				DistDir: queue.DistDir,
				Detail: fmt.Sprintf("%s: %s", prefix, err.Error()),
			}
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
			
			nameMap := make(map[string]string)
			for _, file := range files {
				generatedName := utils.GenerateUniqueRandomName(nameMap)
				nameMap[generatedName] = file.Name()
				
				if file.IsDir() {
					err = os.MkdirAll(filepath.Join(queue.DistDir, generatedName), 0755)
					if err != nil {
						errHandler("Failed to create directory", err)
						return
					}
					
					dispatcherQueue <- dispatcherMessage{
						MsgType: FIND_DIR,
						SrcDir: filepath.Join(queue.SrcDir, file.Name()),
						DistDir: filepath.Join(queue.DistDir, generatedName),
						Detail: "",
					}
				} else {
					content, err := os.ReadFile(filepath.Join(queue.SrcDir, file.Name()))
					if err != nil {
						errHandler("Failed to read file", err)
						return
					}
					
					archive, err := data.ToArchiveData(file.Name(), content, password)
					if err != nil {
						errHandler("Failed to create archive data", err)
						return
					}
					
					err = archive.Export(filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", generatedName)))
					if err != nil {
						errHandler("Failed to export archive", err)
						return
					}
					
					//fmt.Printf("Successfully archived %s -> %s\n", file.Name(), filepath.Join(queue.DistDir, generatedName))
				}
			}
			
			dispatcherQueue <- dispatcherMessage{
				MsgType: FINISH_JOB,
				SrcDir: queue.SrcDir,
				DistDir: queue.DistDir,
				Detail: "",
			}
		}()
	}
}
