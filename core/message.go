package core


// ディスパッチャが扱うメッセージの種類。
type workerToManagerMessageType string

const (
	FIND_DIR   workerToManagerMessageType = "FIND_DIR"   // 処理対象ディレクトリの通知
	FINISH_JOB workerToManagerMessageType = "FINISH_JOB" // ジョブ完了通知
	ERROR      workerToManagerMessageType = "ERROR"      // エラー報告
)

// ワーカーが受け取るメッセージの種類。
type managerToWorkerMessageType string

const (
	NEXT_JOB managerToWorkerMessageType = "NEXT_JOB" // 次の処理ジョブ
	EXIT     managerToWorkerMessageType = "EXIT"     // ワーカー終了指示
)

// ディスパッチャに渡すメッセージ。
type messageFromWorkerToManager struct {
	WorkerId uint
	MsgType  workerToManagerMessageType
	SrcDir   string
	DistDir  string
	Detail   string
}

// ワーカーに渡すジョブまたは終了指示。
type messageFromManagerToWorker struct {
	MsgType managerToWorkerMessageType
	SrcDir  string
	DistDir string
	Detail  string
}
