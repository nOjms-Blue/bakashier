package core

import (
	"runtime"
	"sync"
)


// srcDir を暗号化・圧縮して distDir にバックアップする。
// ディスパッチャ1つとワーカー4つを起動し、チャネルでジョブを分配する。
func Backup(srcDir string, distDir string, password string, chunkSize uint64, limit Limit) {
	var wg sync.WaitGroup
	var workers int = runtime.GOMAXPROCS(0)
	var queueSize int = workers * 8
	
	if workers <= 0 {
		workers = 1
		queueSize = 8
	}
	
	dispatcherQueue := make(chan dispatcherMessage, queueSize)
	workerQueue := make(chan workerMessage, queueSize)
	
	dispatcherQueue <- dispatcherMessage{
		MsgType: FIND_DIR,
		SrcDir: srcDir,
		DistDir: distDir,
		Detail: "",
	}
	
	wg.Add(workers + 1)
	go backupDispatcher(workers, dispatcherQueue, workerQueue, &wg)
	for i := uint(0); i < uint(workers); i++ {
		go backupWorker(i + 1, password, dispatcherQueue, workerQueue, &wg, chunkSize, limit)
	}
	wg.Wait()
	
	close(dispatcherQueue)
	close(workerQueue)
}

// srcDir（バックアップ先）から distDir へ復元する。
// ディスパッチャ1つとワーカー4つを起動し、チャネルでジョブを分配する。
func Restore(srcDir string, distDir string, password string, limit Limit) {
	var wg sync.WaitGroup
	var workers int = runtime.GOMAXPROCS(0)
	var queueSize int = workers * 8
	
	if workers <= 0 {
		workers = 1
		queueSize = 8
	}
	
	dispatcherQueue := make(chan dispatcherMessage, queueSize)
	workerQueue := make(chan workerMessage, queueSize)
	
	dispatcherQueue <- dispatcherMessage{
		MsgType: FIND_DIR,
		SrcDir:  srcDir,
		DistDir: distDir,
		Detail:  "",
	}
	
	wg.Add(workers + 1)
	go restoreDispatcher(workers, dispatcherQueue, workerQueue, &wg)
	for i := uint(0); i < uint(workers); i++ {
		go restoreWorker(i + 1, password, dispatcherQueue, workerQueue, &wg, limit)
	}
	wg.Wait()
	
	close(dispatcherQueue)
	close(workerQueue)
}
