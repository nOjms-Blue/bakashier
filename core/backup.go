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
)


// ディスパッチャキューからメッセージを受け取り、ワーカーにジョブを配分する。
// FIND_DIR でジョブを投入し、全ジョブが FINISH_JOB で完了すると各ワーカーに EXIT を送る。
func backupDispatcher(workers int, dispatcherQueue <-chan dispatcherMessage, workerQueue chan<- workerMessage, wg *sync.WaitGroup) {
	defer wg.Done()
	
	var untreated int = 0
	var untreatedMessage = []workerMessage{}
	for {
		msg := <-dispatcherQueue
		
		switch msg.MsgType {
		case FIND_DIR:
			untreatedMessage = append(untreatedMessage, workerMessage{
				MsgType: NEXT_JOB,
				SrcDir: msg.SrcDir,
				DistDir: msg.DistDir,
				Detail: msg.Detail,
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
					SrcDir: "",
					DistDir: "",
					Detail: "",
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

// ワーカーキューからジョブを受け取り、ディレクトリを走査してファイルをアーカイブする。
// 既存の _directory_.bks を読み、変更のないファイルはスキップする。ディレクトリは FIND_DIR で再投入する。
func backupWorker(password string, dispatcherQueue chan<- dispatcherMessage, workerQueue <-chan workerMessage, wg *sync.WaitGroup, chunkSize uint64, limit Limit) {
	defer wg.Done()
	var processedSize uint64 = 0
	
	for {
		queue := <-workerQueue
		if queue.MsgType == EXIT { break }
		
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
					
					// 子ディレクトリの発見を通知
					dispatcherQueue <- dispatcherMessage{
						MsgType: FIND_DIR,
						SrcDir: filepath.Join(queue.SrcDir, file.Name()),
						DistDir: filepath.Join(queue.DistDir, hideName),
						Detail: "",
					}
					
					fmt.Printf("Found directory: %c%s%c\n", '"', filepath.Join(queue.SrcDir, file.Name()), '"')
				} else {
					isNotChangeFile := false
					fileInfo, err := file.Info()
					if err != nil {
						errHandler("Failed to get file info", err)
						continue
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
						continue
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
					fmt.Printf("File archived: %c%s%c -> %c%s%c\n", '"', filepath.Join(queue.SrcDir, file.Name()), '"', '"', filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", hideName)), '"')
					
					if limit.Size > 0 && limit.Wait > 0 {
						processedSize += uint64(fileInfo.Size())
						if limit.Size > 0 && processedSize >= limit.Size {
							time.Sleep(time.Duration(limit.Wait) * time.Second)
							processedSize = processedSize - limit.Size
						}
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
							fmt.Printf("File deleted: %c%s%c\n", '"', filepath.Join(queue.DistDir, fmt.Sprintf("%s.bks", entry.HideName)), '"')
						} else {
							os.RemoveAll(filepath.Join(queue.DistDir, entry.HideName))
							fmt.Printf("Directory deleted: %c%s%c\n", '"', filepath.Join(queue.DistDir, entry.HideName), '"')
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
				
				fmt.Printf("Successfully directory archived: %c%s%c -> %c%s%c\n", '"', queue.SrcDir, '"', '"', queue.DistDir, '"')
			} else {
				fmt.Printf("No changes found in directory: %c%s%c\n", '"', queue.SrcDir, '"')
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
