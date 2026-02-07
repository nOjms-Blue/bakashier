package core

import (
	"sync"
)

func Backup(srcDir string, distDir string, password string) {
	var wg sync.WaitGroup
	dispatcherQueue := make(chan dispatcherMessage, 64)
	workerQueue := make(chan workerMessage, 64)
	
	dispatcherQueue <- dispatcherMessage{
		MsgType: FIND_DIR,
		SrcDir: srcDir,
		DistDir: distDir,
		Detail: "",
	}
	var workers int = 4
	
	wg.Add(workers + 1)
	go backupDispatcher(workers, dispatcherQueue, workerQueue, &wg)
	for i := 0; i < workers; i++ {
		go backupWorker(password, dispatcherQueue, workerQueue, &wg)
	}
	wg.Wait()
	
	close(dispatcherQueue)
	close(workerQueue)
}