package core

import (
	"bakashier/data"
	"os"
	"path/filepath"
)


type MovedDir struct {
	HideName       string
	BeforeRealName string
	AfterRealName  string
}

func checkMovedDirs(srcDir string, distDir string, entries []data.DirectoryEntry, files []os.DirEntry, password string) ([]MovedDir, error) {
	remainEntries := make([]data.DirectoryEntry, len(entries))
	copy(remainEntries, entries)
	remainFiles := make([]os.DirEntry, len(files))
	possibleRemainFiles := []os.DirEntry{}
	copy(remainFiles, files)
	
	// ディレクトリ以外のエントリを除外
	onlyDirEntries := []data.DirectoryEntry{}
	for len(remainEntries) > 0 {
		entry := remainEntries[0]
		remainEntries = remainEntries[1:]
		
		if entry.Type == data.Directory {
			onlyDirEntries = append(onlyDirEntries, entry)
		}
	}
	remainEntries = onlyDirEntries
	
	// 移動していないものを除外する
	for len(remainFiles) > 0 {
		file := remainFiles[0]
		remainFiles = remainFiles[1:]
		
		// ディレクトリ以外を除外
		if !file.IsDir() { continue }
		
		// 移動していないものを除外
		isPossibility := true
		for index, entry := range remainEntries {
			if file.Name() == entry.RealName {
				remainEntries = append(remainEntries[:index], remainEntries[index+1:]...)
				isPossibility = false
				break
			}
		}
		if !isPossibility { continue }
		
		possibleRemainFiles = append(possibleRemainFiles, file)
	}
	remainFiles = possibleRemainFiles
	
	// 移動した可能性のあるフォルダの情報を取得
	possiblesInDirs := map[string]([]os.DirEntry){}
	for _, file := range remainFiles {
		if !file.IsDir() { continue }
		
		name := file.Name()
		path := filepath.Join(srcDir, name)
		possibles, err := os.ReadDir(path)
		if err != nil { continue }
		
		possiblesInDirs[name] = possibles
	}
	
	// 移動した可能性のあるフォルダエントリの情報を取得
	possiblesInEntries := map[string]([]data.DirectoryEntry){}
	for _, entry := range remainEntries {
		path := filepath.Join(distDir, entry.HideName, "_directory_.bks")
		possibles, err := loadDirectoryEntries(path, password)
		if err != nil { continue }
		
		possiblesInEntries[entry.HideName] = possibles
	}
	
	// ディレクトリ内にあるファイル名の一致率を計算
	calcSamePercent := func(entries []data.DirectoryEntry, files []os.DirEntry) float64 {
		// 外側の remainFiles を書き換えないよう、ローカルのコピーに対して処理する
		candidateFiles := make([]os.DirEntry, len(files))
		copy(candidateFiles, files)
		
		count := 0
		for _, entry := range entries {
			sameNameIndex := -1
			for index, file := range candidateFiles {
				if entry.RealName == file.Name() {
					sameNameIndex = index
					count = count + 2
					break
				}
			}
			
			if sameNameIndex < 0 { continue }
			candidateFiles = append(candidateFiles[:sameNameIndex], candidateFiles[sameNameIndex+1:]...)
		}
		
		if count <= 0 { return 0 }
		return float64(len(entries) + len(files)) / float64(count)
	}
	
	// 移動したものかどうかを決定する
	moved := []MovedDir{}
	for _, file := range remainFiles {
		possiblesInDir, ok := possiblesInDirs[file.Name()]
		if !ok { continue }
		
		decideKey := ""
		decidePercent := float64(0)
		for key, entries := range possiblesInEntries {
			percent := calcSamePercent(entries, possiblesInDir)
			if decideKey == "" || percent > decidePercent {
				decideKey = key
				decidePercent = percent
			}
		}
		if decidePercent < 0.5 { continue }
		
		for _, entry := range entries {
			if entry.Type != data.Directory { continue }
			if decideKey == entry.HideName {
				moved = append(moved, MovedDir{
					HideName: entry.HideName,
					BeforeRealName: entry.RealName,
					AfterRealName: file.Name(),
				})
			}
		}
	}
	
	return moved, nil
}
