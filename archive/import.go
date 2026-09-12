package archive

import (
	"bakashier/utils"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// インポート用の暗号解除・展開処理
func importProcess(chunk []byte, hash []byte, password string) ([]byte, error) {
	var err error = nil
	chunk, err = utils.DecryptBytesWithPassword(chunk, password)
	if err != nil {
		return []byte{}, err
	}
	chunk, err = utils.DecompressBytes(chunk)
	if err != nil {
		return []byte{}, err
	}

	// CRC32 ハッシュを検証
	if !bytes.Equal(hash, utils.CRC32HashBytes(chunk)) {
		return []byte{}, errors.New("chunk CRC32 hash mismatch")
	}
	return chunk, nil
}

func (bks BksArchive) Import(reader io.Reader, getWriter func(name string) (io.Writer, error)) error {
	password := bks.Password

	// ヘッダの最小部分の取得
	header := [9]byte{0, 0, 0, 0, 0, 0, 0, 0, 0}
	_, err := io.ReadFull(reader, header[:])
	if err != nil {
		return err
	}

	// 形式判定
	if header[0] != byte('B') || header[1] != byte('K') || header[2] != byte('S') {
		return errors.New("file is not a valid archived file")
	}

	// 対応バージョンのチェック
	if binary.BigEndian.Uint16(header[3:5]) != 1 {
		return errors.New("unsupported version number")
	}

	// ファイル名情報の取得
	chunkLength32 := binary.BigEndian.Uint32(header[5:9])
	if uint64(chunkLength32) > maxEncryptedNameSize {
		return fmt.Errorf("invalid archive: name block too large (%d bytes)", chunkLength32)
	}
	chunk := make([]byte, chunkLength32)
	_, err = io.ReadFull(reader, chunk)
	if err != nil {
		return err
	}
	chunkCRC := make([]byte, 4)
	_, err = io.ReadFull(reader, chunkCRC)
	if err != nil {
		return err
	}
	originalBytes, err := importProcess(chunk, chunkCRC, password)
	if err != nil {
		return err
	}
	name := string(originalBytes)

	// writer の取得。名前の安全性（zip-slip）はファイルへ書き出す呼び出し側で検証する。
	// _directory_.bks はソースパスを名前として格納するため、ここでは拒否しない。
	writer, err := getWriter(name)
	if err != nil {
		return fmt.Errorf("failed to get writer: %w", err)
	}

	isWrote := false
	chunkLenBytes := make([]byte, 8)
	chunkLength := uint64(0)
	for {
		n, err := io.ReadFull(reader, chunkLenBytes)
		if n == 0 || err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if n != 8 {
			return errors.New("invalid archive: truncated chunk length")
		}
		chunkLength = binary.BigEndian.Uint64(chunkLenBytes)
		if chunkLength > maxEncryptedChunkSize {
			return fmt.Errorf("invalid archive: chunk too large (%d bytes)", chunkLength)
		}

		// チャンクを読み込む
		chunk = make([]byte, chunkLength)
		n, err = io.ReadFull(reader, chunk)
		if err != nil {
			return err
		}
		if n != int(chunkLength) {
			return errors.New("invalid archive: truncated chunk")
		}

		// CRC32 ハッシュを読み込む
		chunkCRC = make([]byte, 4)
		n, err = io.ReadFull(reader, chunkCRC)
		if err != nil {
			return err
		}
		if n != 4 {
			return errors.New("invalid archive: truncated chunk hash")
		}

		// チャンクを復号・展開
		chunk, err = importProcess(chunk, chunkCRC, password)
		if err != nil {
			return err
		}

		if _, err = writer.Write(chunk); err != nil {
			return err
		}
		isWrote = true
	}

	if !isWrote {
		if _, err = writer.Write([]byte{}); err != nil {
			return err
		}
	}

	return nil
}
