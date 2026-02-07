package core

import (
	"bakashier/data"

	"os"
	"path/filepath"
)

// カレントディレクトリのパスと ArchiveData とパスワードを受け取り、名前を取り出してカレントディレクトリ配下にその名前のディレクトリを作成する。
func MakeDirectory(currentDir string, archive data.ArchiveData, password string) error {
	filename, _, err := data.FromArchiveData(archive, password)
	if err != nil { return err }
	dirPath := filepath.Join(currentDir, filename)
	return os.Mkdir(dirPath, 0755)
}

// ディレクトリパスとパスワードを受け取り、data.ArchiveData を作成する。
// Data にはそのディレクトリ直下のファイル・サブディレクトリの情報（名前・サイズ・最終更新日時）を格納する。
func MakeArchiveDirectory(dirPath string, password string) (data.ArchiveData, error) {
	dirName := filepath.Base(dirPath)
	return data.ToArchiveData(dirName, []byte{}, password)
}
