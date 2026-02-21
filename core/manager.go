package core

import (
	"runtime"
	"sync"
	
	"bakashier/view"
)


// srcDir を暗号化・圧縮して distDir にバックアップする。
// ディスパッチャ1つとワーカー4つを起動し、チャネルでジョブを分配する。
func Backup(srcDir string, distDir string, password string, chunkSize uint64, limit Limit, toViewQueue chan<- view.MessageToView, fromViewQueue <-chan view.MessageToDispatcher) {
	var wg sync.WaitGroup
	var workers int = runtime.GOMAXPROCS(0)
	var queueSize int = workers * 8
	
	if workers <= 0 {
		workers = 1
		queueSize = 8
	}
	
	workerToDispatcherQueue := make(chan dispatcherMessage, queueSize)
	dispatcherToWorkerQueue := make(chan workerMessage, queueSize)
	
	workerToDispatcherQueue <- dispatcherMessage{
		MsgType: FIND_DIR,
		SrcDir: srcDir,
		DistDir: distDir,
		Detail: "",
	}
	
	wg.Add(workers + 1)
	go backupDispatcher(workers, workerToDispatcherQueue, dispatcherToWorkerQueue, toViewQueue, fromViewQueue, &wg)
	for i := uint(0); i < uint(workers); i++ {
		go backupWorker(i + 1, password, workerToDispatcherQueue, dispatcherToWorkerQueue, toViewQueue, &wg, chunkSize, limit)
	}
	wg.Wait()
	
	close(workerToDispatcherQueue)
	close(dispatcherToWorkerQueue)
}

// srcDir（バックアップ先）から distDir へ復元する。
// ディスパッチャ1つとワーカー4つを起動し、チャネルでジョブを分配する。
func Restore(srcDir string, distDir string, password string, limit Limit, toViewQueue chan<- view.MessageToView, fromViewQueue <-chan view.MessageToDispatcher) {
	var wg sync.WaitGroup
	var workers int = runtime.GOMAXPROCS(0)
	var queueSize int = workers * 8
	
	if workers <= 0 {
		workers = 1
		queueSize = 8
	}
	
	workerToDispatcherQueue := make(chan dispatcherMessage, queueSize)
	dispatcherToWorkerQueue := make(chan workerMessage, queueSize)
	
	workerToDispatcherQueue <- dispatcherMessage{
		MsgType: FIND_DIR,
		SrcDir:  srcDir,
		DistDir: distDir,
		Detail:  "",
	}
	
	wg.Add(workers + 1)
	go restoreDispatcher(workers, workerToDispatcherQueue, dispatcherToWorkerQueue, toViewQueue, fromViewQueue, &wg)
	for i := uint(0); i < uint(workers); i++ {
		go restoreWorker(i + 1, password, workerToDispatcherQueue, dispatcherToWorkerQueue, toViewQueue, &wg, limit)
	}
	wg.Wait()
	
	close(workerToDispatcherQueue)
	close(dispatcherToWorkerQueue)
}
