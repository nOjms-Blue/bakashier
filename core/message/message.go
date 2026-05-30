package message


type MessageCategory string
const (
	MSG_CAT_INST_SEARCH     MessageCategory = "INST_SEARCH"      // フォルダの検索指示
	MSG_CAT_INST_BACKUP_V1  MessageCategory = "INST_BACKUP_V1"   // bks へバックアップ指示
	MSG_CAT_INST_RESTORE_V1 MessageCategory = "INST_RESTORE_V1"  // bks からリストア指示
	MSG_CAT_INST_BACKUP_V2  MessageCategory = "INST_BACKUP_V2"   // bks へバックアップ指示
	MSG_CAT_INST_RESTORE_V2 MessageCategory = "INST_RESTORE_V2"  // bks からリストア指示
	MSG_CAT_INST_FILE_HASH  MessageCategory = "INST_FILE_HASH"   // ファイルハッシュを求める指示
	MSG_CAT_INST_VERIFY     MessageCategory = "INST_VERIFY"      // bks の整合性確認指示
	MSG_CAT_INST_FINISH     MessageCategory = "INST_FINISH"      // 終了支持
	MSG_CAT_INST_REGISTER   MessageCategory = "INST_REGISTER"    // bks を登録
	
	MSG_CAT_REPO_ERROR      MessageCategory = "REPO_ERROR"       // エラー報告
	MSG_CAT_REPO_FOUND_FILE MessageCategory = "REPO_FOUND_FILE"  // ファイルの発見報告
	MSG_CAT_REPO_FOUND_DIR  MessageCategory = "REPO_FOUND_DIR"   // ディレクトリの発見報告
	MSG_CAT_REPO_RESTORE_V1 MessageCategory = "REPO_RESTORE_V1"  // bks からリストア結果の報告
	MSG_CAT_REPO_HASH       MessageCategory = "REPO_HASH"        // ハッシュ値の計算結果の報告
	MSG_CAT_REPO_VERIFIED   MessageCategory = "REPO_VERIFIED"    // bks の整合性確認結果の報告
	MSG_CAT_REPO_REGISTERED MessageCategory = "REPO_REGISTERED"  // bks の登録完了の報告
)

type Message struct {
	FromId   uint32           // 送信元ID (0 = manager, 1~ = worker)
	Category MessageCategory  // カテゴリ
	SrcPath  string           // 元パス
	DstPath  string           // 先パス
	Data     any              // データ
}
