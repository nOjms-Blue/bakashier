package worker

import (
	"bakashier/core/message"
	"os"
	"path/filepath"
)


func searchTask(msg message.Message, ch chan<- message.Message) error {
	entries, err := os.ReadDir(msg.SrcPath)
	if err != nil { return err }
	
	for _, entry := range entries {
		if entry.IsDir() {
			ch <- message.Message{
				Category: message.MSG_CAT_REPO_FOUND_DIR,
				SrcPath: filepath.Join(msg.SrcPath, entry.Name()),
			}
		} else {
			ch <- message.Message{
				Category: message.MSG_CAT_REPO_FOUND_FILE,
				SrcPath: filepath.Join(msg.SrcPath, entry.Name()),
			}
		}
	}
	
	return nil
}
