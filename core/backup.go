package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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
			newEntries := make(map[string]data.DirectoryEntry)
			directoryEntryFile := filepath.Join(queue.DistDir, "_directory_.bks")
			entries, err := loadDirectoryEntries(directoryEntryFile, password)
			if err != nil {
				errHandler("Failed to load directory entries", err)
				return
			}
			
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
					// ディレクトリを作成
					err = os.MkdirAll(filepath.Join(queue.DistDir, hideName), 0755)
					if err != nil {
						errHandler("Failed to create directory", err)
						return
					}
					
					// ディレクトリエントリを追加
					newEntries[hideName] = data.DirectoryEntry{
						Type: data.Directory,
						RealName: file.Name(),
						HideName: hideName,
						Size: uint64(0),
						ModTime: time.Now(),
					}
					
					// 子ディレクトリの発見を通知
					dispatcherQueue <- dispatcherMessage{
						MsgType: FIND_DIR,
						SrcDir: filepath.Join(queue.SrcDir, file.Name()),
						DistDir: filepath.Join(queue.DistDir, hideName),
						Detail: "",
					}
					
					fmt.Printf("Successfully found directory %s\n", file.Name())
				} else {
					fileInfo, err := file.Info()
					if err != nil {
						errHandler("Failed to get file info", err)
						continue
					}
					
					// 変更がない場合はスキップ
					if (entry.Type == data.File) {
						if (entry.Size == uint64(fileInfo.Size()) && entry.ModTime.Equal(fileInfo.ModTime())) {
							newEntries[hideName] = entry
							continue
						}
					}
					
					// ファイルを読み込む
					content, err := os.ReadFile(filepath.Join(queue.SrcDir, file.Name()))
					if err != nil {
						errHandler("Failed to read file", err)
						return
					}
					
					// アーカイブデータを作成
					archive, err := data.ToArchiveData(file.Name(), content, password)
					if err != nil {
						errHandler("Failed to create archive data", err)
						return
					}
					
					// アーカイブデータを保存
					err = archive.Export(filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", hideName)))
					if err != nil {
						errHandler("Failed to export archive", err)
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
					fmt.Printf("Successfully archived %s -> %s\n", file.Name(), filepath.Join(queue.DistDir, hideName))
				}
			}
			
			// ディレクトリエントリを保存
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
		}()
		
		dispatcherQueue <- dispatcherMessage{
			MsgType: FINISH_JOB,
			SrcDir: queue.SrcDir,
			DistDir: queue.DistDir,
			Detail: "",
		}
	}
}
