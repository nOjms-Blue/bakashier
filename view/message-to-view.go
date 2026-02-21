package view


type SourceType string
const (
	WORKER SourceType = "WORKER"
	MANAGER SourceType = "MANAGER"
)

type MessageToViewType string
const (
	ADD_WORKER MessageToViewType = "ADD_WORKER"   // ワーカー追加
	START_DIR MessageToViewType = "START_DIR"     // ディレクトリ処理開始
	START_FILE MessageToViewType = "START_FILE"   // ファイル処理開始
	FINISH_FILE MessageToViewType = "FINISH_FILE" // ファイル処理完了
	FINISH_DIR MessageToViewType = "FINISH_DIR"   // ディレクトリ処理完了
	ERROR MessageToViewType = "ERROR"             // エラー報告
	FINISHED MessageToViewType = "FINISHED"       // 処理完了
)

type MessageToView struct {
	Source   SourceType         // メッセージの送信元
	MsgType  MessageToViewType  // メッセージの種類
	WorkerId uint               // ワーカーID
	SrcPath  string             // ソースパス
	DistPath string             // 宛先パス
	Detail   string             // メッセージの詳細
}
