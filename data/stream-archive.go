package data

import (
	"io"
	"os"
	"path/filepath"
)


var ChunkSize uint64 = 16 * 1024 * 1024 // 16MB

func ExportStreamArchive(srcFile string, dstFile string, fileName string, password string, chunkSize uint64) error {
	// ソースファイルのサイズを取得
	fileInfo, err := os.Stat(srcFile)
	if err != nil { return err }
	remainFileSize := uint64(fileInfo.Size())
	
	// ソースファイルを開く
	src, err := os.Open(srcFile)
	if err != nil { return err }
	defer src.Close()
	
	// 書き出し先ファイルを開く
	dst, err := os.Create(dstFile)
	if err != nil { return err }
	defer dst.Close()

	ExportBks(
		fileName,
		func(length uint64) ([]byte, error) {
			if remainFileSize == 0 { return []byte{}, nil }
			if remainFileSize < length { length = remainFileSize }
			remainFileSize = remainFileSize - length
			
			chunk := make([]byte, length)
			n, err := src.Read(chunk)
			if err == io.EOF || n == 0 { return []byte{}, nil }
			if err != nil { return []byte{}, err }
			return chunk, nil
		},
		func(data []byte) error {
			_, err := dst.Write(data)
			return err
		},
		password,
		chunkSize,
	)
	
	return nil
}

func ImportStreamArchive(archiveFile string, dstDirectory string, password string) (string, error) {
	var dstFile string = ""
	var dst *os.File = nil
	
	// アーカイブファイルを開く
	archive, err := os.Open(archiveFile)
	if err != nil { return "", err }
	defer archive.Close()
	
	err = ImportBks(
		func(length uint64) ([]byte, error) {
			chunk := make([]byte, length)
			n, err := archive.Read(chunk)
			if err == io.EOF || n == 0 { return []byte{}, nil }
			if err != nil { return []byte{}, err }
			return chunk, nil
		},
		func(name string, data []byte) error {
			var err error = nil
			if dst == nil {
				dstFile = filepath.Join(dstDirectory, name)
				dst, err = os.Create(dstFile)
				if err != nil { return err }
			}
			_, err = dst.Write(data)
			return err
		},
		password,
	)
	if dst != nil { dst.Close() }
	
	return dstFile, nil
}

func ImportStreamArchiveV2(archiveFile string, dstFile string, password string) error {
	// アーカイブファイルを開く
	archive, err := os.Open(archiveFile)
	if err != nil { return err }
	defer archive.Close()
	
	// 出力先ファイルを開く
	dst, err := os.Create(dstFile)
	if err != nil { return err }
	defer dst.Close()
	
	err = ImportBks(
		func(length uint64) ([]byte, error) {
			chunk := make([]byte, length)
			n, err := archive.Read(chunk)
			if err == io.EOF || n == 0 { return []byte{}, nil }
			if err != nil { return []byte{}, err }
			return chunk, nil
		},
		func(name string, data []byte) error {
			var err error = nil
			_, err = dst.Write(data)
			return err
		},
		password,
	)
	
	return nil
}
