package data

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)


var ChunkSize uint64 = 16 * 1024 * 1024 // 16MB

// アーカイブ由来の名前が、ディレクトリ要素を含まない安全な単一のファイル名かを検証する。
// パストラバーサル（zip-slip）対策として、リストア時の書き出し先の組み立て前に使用する。
func IsSafeFileName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, "/\\") || strings.ContainsRune(name, 0) {
		return false
	}
	if name != filepath.Base(name) {
		return false
	}
	return true
}

// ファイルから最大 length バイトを読み込む。io.ReadFull により部分読み込み（short read）でも
// 実際に読めたバイト数だけを返し、ゼロ埋めデータが混入しないことを保証する。
func readChunkFull(r io.Reader, length uint64) ([]byte, error) {
	if length == 0 { return []byte{}, nil }
	chunk := make([]byte, length)
	n, err := io.ReadFull(r, chunk)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return chunk[:n], nil
	}
	if err != nil { return []byte{}, err }
	return chunk, nil
}

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

	err = ExportBks(
		fileName,
		func(length uint64) ([]byte, error) {
			if remainFileSize == 0 { return []byte{}, nil }
			if remainFileSize < length { length = remainFileSize }
			
			chunk, err := readChunkFull(src, length)
			if err != nil { return []byte{}, err }
			remainFileSize = remainFileSize - uint64(len(chunk))
			return chunk, nil
		},
		func(data []byte) error {
			_, err := dst.Write(data)
			return err
		},
		password,
		chunkSize,
	)
	if err != nil {
		// 失敗時は不完全なアーカイブを残さない
		dst.Close()
		os.Remove(dstFile)
		return err
	}
	
	return dst.Close()
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
			return readChunkFull(archive, length)
		},
		func(name string, data []byte) error {
			var err error = nil
			if dst == nil {
				if !IsSafeFileName(name) {
					return fmt.Errorf("unsafe file name in archive: %q", name)
				}
				dstFile = filepath.Join(dstDirectory, name)
				dst, err = os.Create(dstFile)
				if err != nil { return err }
			}
			_, err = dst.Write(data)
			return err
		},
		password,
	)
	if dst != nil {
		closeErr := dst.Close()
		if err == nil { err = closeErr }
	}
	if err != nil {
		// 失敗時は不完全な出力ファイルを残さない
		if dstFile != "" { os.Remove(dstFile) }
		return "", err
	}
	if dstFile == "" {
		return "", errors.New("archive did not contain any file data")
	}
	
	return dstFile, nil
}
