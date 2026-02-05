package core

import (
	"bakashier/data"

	"io"
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

// ファイルパスとパスワードを受け取り、data.ArchiveDataを作成する関数
func MakeArchiveFile(filePath string, password string) (data.ArchiveData, error) {
	filename := filepath.Base(filePath)
	f, err := os.Open(filePath)
	if err != nil {
		return data.ArchiveData{}, err
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return data.ArchiveData{}, err
	}

	return data.ToArchiveData(filename, content, password)
}