package data

import (
	"errors"
	"os"
)


const SIMPLE_ARCHIVE_MAX_SIZE = 4 * 1024 * 1024 * 1024 // 4GiB

func ImportAll(fileName string, password string) (string, []byte, error) {
	var parsedName = ""
	var parsedContent = []byte{}
	
	// ファイルサイズが 4GB を超える場合はエラーを返す。
	fileInfo, err := os.Stat(fileName)
	if err != nil { return "", []byte{}, err }
	if fileInfo.Size() > SIMPLE_ARCHIVE_MAX_SIZE { return "", []byte{}, errors.New("file is too large") }
	
	// ファイルの読み込み
	content, err := os.ReadFile(fileName)
	if err != nil { return "", []byte{}, err }
	
	// 圧縮・暗号化されたデータから取り込む
	offset := uint64(0)
	err = ImportBks(
		func(length uint64) ([]byte, error) {
			if offset >= uint64(len(content)) { return []byte{}, nil }
			if offset + length > uint64(len(content)) { length = uint64(len(content)) - offset }
			b := content[offset:offset+length]
			offset = offset + length
			return b, nil
		},
		func(name string, data []byte) error {
			parsedName = name
			parsedContent = append(parsedContent, data...)
			return nil
		},
		password,
	)
	if err != nil {
		return "", []byte{}, err
	}
	
	return parsedName, parsedContent, nil
}

func ExportAll(exportFileName string, fileName string, content []byte, password string, chunkSize uint64) error {
	// 出力先ファイルを開く
	fp, err := os.Create(exportFileName)
	if err != nil { return err }
	defer fp.Close()
	
	// データを暗号化・圧縮処理をして書き込み
	offset := uint64(0)
	err = ExportBks(
		fileName,
		func(length uint64) ([]byte, error) {
			if offset >= uint64(len(content)) { return []byte{}, nil }
			if offset + length > uint64(len(content)) { length = uint64(len(content)) - offset }
			b := content[offset:offset+length]
			offset = offset + length
			return b, nil
		},
		func(data []byte) error {
			_, err := fp.Write(data)
			return err
		},
		password,
		chunkSize,
	)
	if err != nil { return err }
	
	return nil
}

func ExportAllV2(exportFileName string, content []byte, password string, chunkSize uint64) error {
	return ExportAll(exportFileName, "", content, password, chunkSize)
}
