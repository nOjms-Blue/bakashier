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

// ディレクトリ名とパスワードを受け取り、data.ArchiveDataを作成する関数
func MakeArchiveDirectory(dirName string, password string) (data.ArchiveData, error) {
	// ディレクトリなので内容は空のバイト配列
	content := []byte{}
	return data.ToArchiveData(dirName, content, password)
}
