package core

import (
	"sync"
)

// srcDir を暗号化・圧縮して distDir にバックアップする。
// ディスパッチャ1つとワーカー4つを起動し、チャネルでジョブを分配する。
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

// srcDir（バックアップ先）から distDir へ復元する。
// ディスパッチャ1つとワーカー4つを起動し、チャネルでジョブを分配する。
func Restore(srcDir string, distDir string, password string) {
	var wg sync.WaitGroup
	dispatcherQueue := make(chan dispatcherMessage, 64)
	workerQueue := make(chan workerMessage, 64)
	
	dispatcherQueue <- dispatcherMessage{
		MsgType: FIND_DIR,
		SrcDir:  srcDir,
		DistDir: distDir,
		Detail:  "",
	}
	var workers int = 4
	
	wg.Add(workers + 1)
	go restoreDispatcher(workers, dispatcherQueue, workerQueue, &wg)
	for i := 0; i < workers; i++ {
		go restoreWorker(password, dispatcherQueue, workerQueue, &wg)
	}
	wg.Wait()
	
	close(dispatcherQueue)
	close(workerQueue)
}
