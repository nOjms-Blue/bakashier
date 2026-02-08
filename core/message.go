package core

type dispatcherMessageType string
const (
	FIND_DIR    dispatcherMessageType = "FIND_DIR"
	FINISH_JOB  dispatcherMessageType = "FINISH_JOB"
	ERROR       dispatcherMessageType = "ERROR"
)

type workerMessageType string
const (
	NEXT_JOB    workerMessageType = "NEXT_JOB"
	EXIT        workerMessageType = "EXIT"
)

type dispatcherMessage struct {
	MsgType dispatcherMessageType
	SrcDir  string
	DistDir string
	Detail  string
}

type workerMessage struct {
	MsgType workerMessageType
	SrcDir  string
	DistDir string
	Detail  string
}