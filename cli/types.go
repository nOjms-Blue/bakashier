package cli


// アプリケーションの動作モード（バックアップ/復元/バージョン表示）。
type ModeType string

const (
	ModeBackup  ModeType = "backup"
	ModeRestore ModeType = "restore"
	ModeVersion ModeType = "version"
	ModeHelp    ModeType = "help"
)

// コマンドライン引数を解析した結果。
type ParsedArgs struct {
	Mode      ModeType
	SrcDir    string
	DistDir   string
	Password  string
	ChunkSize uint64
}
