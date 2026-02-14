package core

import (
	"os"
	
	"bakashier/data"
)


// _directory_.bks からエントリ一覧を読み込む。ファイルが存在しない場合は空スライスを返す。復号に password を使用する。
func loadDirectoryEntries(directoryEntryFile string, password string) ([]data.DirectoryEntry, error) {
	var entryFile data.ArchiveData
	if _, err := os.Stat(directoryEntryFile); err == nil {
		err = entryFile.Import(directoryEntryFile)
		if err != nil { return []data.DirectoryEntry{}, err }
		_, content, err := data.FromArchiveData(entryFile, password)
		if err != nil { return []data.DirectoryEntry{}, err }
		entries, err := data.ImportDirectoryEntries(content)
		if err != nil { return []data.DirectoryEntry{}, err }
		return entries, nil
	}
	return []data.DirectoryEntry{}, nil
}