package archive

import (
	"bakashier/utils"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

// インポート用の暗号解除・展開処理。
func importProcess(chunk []byte, hash []byte, password string, writer io.Writer, maxSize uint64) error {
	chunk, err := utils.DecryptBytesWithPassword(chunk, password)
	if err != nil {
		return err
	}

	hasher := crc32.NewIEEE()
	if err := utils.DecompressBytesToWriter(chunk, hasher, maxSize); err != nil {
		return err
	}

	// CRC32 ハッシュを検証
	if !bytes.Equal(hash, hasher.Sum(nil)) {
		return errors.New("chunk CRC32 hash mismatch")
	}

	// 整合性確認が完了するまで呼び出し側の writer には書き込まない。
	return utils.DecompressBytesToWriter(chunk, writer, maxSize)
}

func readArchiveBlock(reader io.Reader, length uint64) ([]byte, error) {
	if length > uint64(int(^uint(0)>>1)) {
		return nil, errors.New("archive block is too large for this platform")
	}
	chunk, err := io.ReadAll(io.LimitReader(reader, int64(length)))
	if err != nil {
		return nil, err
	}
	if uint64(len(chunk)) != length {
		return nil, errors.New("invalid archive: truncated block")
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
	chunk, err := readArchiveBlock(reader, uint64(chunkLength32))
	if err != nil {
		return err
	}
	chunkCRC := make([]byte, 4)
	_, err = io.ReadFull(reader, chunkCRC)
	if err != nil {
		return err
	}
	var originalName bytes.Buffer
	err = importProcess(chunk, chunkCRC, password, &originalName, maxEncryptedNameSize)
	if err != nil {
		return err
	}
	name := originalName.String()

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
		chunk, err = readArchiveBlock(reader, chunkLength)
		if err != nil {
			return err
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

		// チャンクを復号し、展開サイズを制限しながら writer へストリーミングする。
		if err = importProcess(chunk, chunkCRC, password, writer, MAX_CHUNK_SIZE); err != nil {
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
