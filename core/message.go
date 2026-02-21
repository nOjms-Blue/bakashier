package core


// ディスパッチャが扱うメッセージの種類。
type workerToDispatcherMessageType string
const (
	FIND_DIR    workerToDispatcherMessageType = "FIND_DIR"   // 処理対象ディレクトリの通知
	FINISH_JOB  workerToDispatcherMessageType = "FINISH_JOB" // ジョブ完了通知
	ERROR       workerToDispatcherMessageType = "ERROR"      // エラー報告
)

// ワーカーが受け取るメッセージの種類。
type dispatcherToWorkerMessageType string
const (
	NEXT_JOB    dispatcherToWorkerMessageType = "NEXT_JOB" // 次の処理ジョブ
	EXIT        dispatcherToWorkerMessageType = "EXIT"     // ワーカー終了指示
)

// ディスパッチャに渡すメッセージ。
type messageFromWorkerToDispatcher struct {
	WorkerId uint
	MsgType workerToDispatcherMessageType
	SrcDir  string
	DistDir string
	Detail  string
}

// ワーカーに渡すジョブまたは終了指示。
type messageFromDispatcherToWorker struct {
	MsgType dispatcherToWorkerMessageType
	SrcDir  string
	DistDir string
	Detail  string
}