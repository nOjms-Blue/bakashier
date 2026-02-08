package core

import (
	"runtime"
	"sync"
)

// srcDir を暗号化・圧縮して distDir にバックアップする。
// ディスパッチャ1つとワーカー4つを起動し、チャネルでジョブを分配する。
func Backup(srcDir string, distDir string, password string) {
	var wg sync.WaitGroup
	var workers int
	var queueSize int
	
	workers = runtime.GOMAXPROCS(0)
	if workers >= 2 { workers = workers - 1 }
	queueSize = workers * 8
	
	dispatcherQueue := make(chan dispatcherMessage, queueSize)
	workerQueue := make(chan workerMessage, queueSize)
	
	dispatcherQueue <- dispatcherMessage{
		MsgType: FIND_DIR,
		SrcDir: srcDir,
		DistDir: distDir,
		Detail: "",
	}
	
	wg.Add(workers + 1)
	go backupDispatcher(workers, queueSize, dispatcherQueue, workerQueue, &wg)
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
	var workers int
	var queueSize int
	
	workers = runtime.GOMAXPROCS(0)
	if workers >= 2 { workers = workers - 1 }
	queueSize = workers * 8
	
	dispatcherQueue := make(chan dispatcherMessage, queueSize)
	workerQueue := make(chan workerMessage, queueSize)
	
	dispatcherQueue <- dispatcherMessage{
		MsgType: FIND_DIR,
		SrcDir:  srcDir,
		DistDir: distDir,
		Detail:  "",
	}
	
	wg.Add(workers + 1)
	go restoreDispatcher(workers, queueSize, dispatcherQueue, workerQueue, &wg)
	for i := 0; i < workers; i++ {
		go restoreWorker(password, dispatcherQueue, workerQueue, &wg)
	}
	wg.Wait()
	
	close(dispatcherQueue)
	close(workerQueue)
}
