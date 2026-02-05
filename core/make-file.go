package core

import (
	"bakashier/data"

	"os"
	"path/filepath"
)

// カレントディレクトリのパスと ArchiveData とパスワードを受け取り、
// 名前と中身を取り出して、カレントディレクトリ配下にその名前のファイルを作成し、
// 復号化した content の []byte を書き込む。
func MakeFile(currentDir string, archive data.ArchiveData, password string) error {
	filename, content, err := data.FromArchiveData(archive, password)
	if err != nil { return err }
	filePath := filepath.Join(currentDir, filename)
	return os.WriteFile(filePath, content, 0644)
}
