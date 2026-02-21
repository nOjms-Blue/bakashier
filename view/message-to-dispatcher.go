package view


type MessageToDispatcherType string
const (
	STOP_WORKERS MessageToDispatcherType = "STOP_WORKERS"      // 一時停止指示
	RESUME_WORKERS MessageToDispatcherType = "RESUME_WORKERS"  // 再開指示
	TERMINATION MessageToDispatcherType = "TERMINATION"        // 終了指示
)

type MessageToDispatcher struct {
	MsgType MessageToDispatcherType  // メッセージの種類
}
