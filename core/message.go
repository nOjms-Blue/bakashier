package core


// ディスパッチャが扱うメッセージの種類。
type dispatcherMessageType string
const (
	FIND_DIR    dispatcherMessageType = "FIND_DIR"   // 処理対象ディレクトリの通知
	FINISH_JOB  dispatcherMessageType = "FINISH_JOB" // ジョブ完了通知
	ERROR       dispatcherMessageType = "ERROR"     // エラー報告
)

// ワーカーが受け取るメッセージの種類。
type workerMessageType string
const (
	NEXT_JOB    workerMessageType = "NEXT_JOB" // 次の処理ジョブ
	EXIT        workerMessageType = "EXIT"     // ワーカー終了指示
)

// ディスパッチャ⇔ワーカー間でやり取りするメッセージ。
type dispatcherMessage struct {
	MsgType dispatcherMessageType
	SrcDir  string
	DistDir string
	Detail  string
}

// ワーカーに渡すジョブまたは終了指示。
type workerMessage struct {
	MsgType workerMessageType
	SrcDir  string
	DistDir string
	Detail  string
}