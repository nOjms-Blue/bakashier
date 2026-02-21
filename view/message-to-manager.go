package view


type MessageToManagerType string

const (
	STOP_WORKERS   MessageToManagerType = "STOP_WORKERS"   // 一時停止指示
	RESUME_WORKERS MessageToManagerType = "RESUME_WORKERS" // 再開指示
	TERMINATION    MessageToManagerType = "TERMINATION"    // 終了指示
)

type MessageToManager struct {
	MsgType MessageToManagerType // メッセージの種類
}
