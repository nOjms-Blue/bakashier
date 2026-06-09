package data

import (
	"bakashier/utils"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)


// エクスポート時に指定できるプレーンテキストチャンクの上限サイズ。
const MAX_CHUNK_SIZE uint64 = 1024 * 1024 * 1024 // 1GiB

// インポート時に許容する圧縮・暗号化済みチャンクの上限サイズ（zlib・AES-GCM のオーバーヘッドを考慮）。
const maxEncryptedChunkSize uint64 = MAX_CHUNK_SIZE + MAX_CHUNK_SIZE/128 + 1024

// インポート時に許容する圧縮・暗号化済み名前ブロックの上限サイズ。
const maxEncryptedNameSize uint64 = 64 * 1024

func ExportBks(name string, reader func(length uint64) ([]byte, error), writer func(data []byte) error, password string, chunkSize uint64) error {
	if chunkSize == 0 || chunkSize > MAX_CHUNK_SIZE {
		return fmt.Errorf("invalid chunk size: %d (must be 1..%d)", chunkSize, MAX_CHUNK_SIZE)
	}
	
	// 出力の圧縮・暗号化処理
	exportProcess := func(chunk []byte) ([]byte, []byte, error) {
		var err error = nil
		
		// CRC32 ハッシュを計算
		chunkCRC := utils.CRC32HashBytes(chunk)
		
		// エクスポート用の処理
		chunk, err = utils.CompressBytes(chunk)
		if err != nil { return []byte{}, []byte{}, err }
		chunk, err = utils.EncryptBytesWithPassword(chunk, password)
		if err != nil { return []byte{}, []byte{}, err }
		
		return chunk, chunkCRC, nil
	}
	
	// ヘッダを書き込む
	var versionBytes = make([]byte, 2)
	var nameLenBytes = make([]byte, 4)
	binary.BigEndian.PutUint16(versionBytes, 1)
	nameBytes, nameCRC, err := exportProcess([]byte(name))
	if err != nil { return err }
	binary.BigEndian.PutUint32(nameLenBytes, uint32(len(nameBytes)))
	if err := writer([]byte("BKS")); err != nil { return err }
	if err := writer(versionBytes); err != nil { return err }
	if err := writer(nameLenBytes); err != nil { return err }
	if err := writer(nameBytes); err != nil { return err }
	if err := writer(nameCRC); err != nil { return err }
	
	chunkLenBytes := make([]byte, 8)
	for {
		chunk, err := reader(chunkSize)
		if err != nil { return err }
		
		// もう次のチャンクがなければ正常終了
		if len(chunk) == 0 { break }
		
		// チャンクを処理する
		chunk, chunkCRC, err := exportProcess(chunk)
		if err != nil { return err }
		
		// チャンク長を書き込む
		binary.BigEndian.PutUint64(chunkLenBytes, uint64(len(chunk)))
		
		if err := writer(chunkLenBytes); err != nil { return err }
		if err := writer(chunk); err != nil { return err }
		if err := writer(chunkCRC); err != nil { return err }
	}
	
	return nil
}

func ImportBks(reader func(length uint64) ([]byte, error), writer func(name string, data []byte) error, password string) error {
	// インポート用の暗号解除・展開処理
	importProcess := func(chunk []byte, hash []byte) ([]byte, error) {
		var err error = nil
		chunk, err = utils.DecryptBytesWithPassword(chunk, password)
		if err != nil { return []byte{}, err }
		chunk, err = utils.DecompressBytes(chunk)
		if err != nil { return []byte{}, err }
		
		// CRC32 ハッシュを検証
		if !bytes.Equal(hash, utils.CRC32HashBytes(chunk)) {
			return []byte{}, errors.New("chunk CRC32 hash mismatch")
		}
		return chunk, nil
	}
	
	// ヘッダの取得
	header, err := reader(9)
	if err != nil { return err }
	if len(header) < 9 {
		return errors.New("file is not a valid archived file (header too short)")
	}
	
	// ヘッダの先頭部分の検証
	if header[0] != byte('B') || header[1] != byte('K') || header[2] != byte('S') {
		return errors.New("file is not a valid archived file")
	}
	if binary.BigEndian.Uint16(header[3:5]) != 1 {
		return errors.New("unsupported version number")
	}
	
	// 名前情報の取得
	nameLen := binary.BigEndian.Uint32(header[5:9])
	if uint64(nameLen) > maxEncryptedNameSize {
		return fmt.Errorf("invalid archive: name block too large (%d bytes)", nameLen)
	}
	nameBytes, err := reader(uint64(nameLen))
	if err != nil { return err }
	if uint64(len(nameBytes)) < uint64(nameLen) {
		return errors.New("invalid archive: truncated name block")
	}
	nameHash, err := reader(4)
	if err != nil { return err }
	if len(nameHash) < 4 {
		return errors.New("invalid archive: truncated name hash")
	}
	nameBytes, err = importProcess(nameBytes, nameHash)
	if err != nil { return err }
	name := string(nameBytes)

	wroteChunk := false
	for {
		chunkLenBytes, err := reader(8)
		if err != nil { return err }
		if len(chunkLenBytes) == 0 { break }
		if len(chunkLenBytes) < 8 {
			return errors.New("invalid archive: truncated chunk length")
		}
		chunkLen := binary.BigEndian.Uint64(chunkLenBytes)
		if chunkLen > maxEncryptedChunkSize {
			return fmt.Errorf("invalid archive: chunk too large (%d bytes)", chunkLen)
		}
		
		// チャンクを読み込む
		chunk, err := reader(chunkLen)
		if err != nil { return err }
		if uint64(len(chunk)) < chunkLen {
			return errors.New("invalid archive: truncated chunk")
		}
		
		// CRC32 ハッシュを読み込む
		chunkCRC, err := reader(4)
		if err != nil { return err }
		if len(chunkCRC) < 4 {
			return errors.New("invalid archive: truncated chunk hash")
		}
		
		// チャンクを復号・展開
		chunk, err = importProcess(chunk, chunkCRC)
		if err != nil { return err }
		
		if err := writer(name, chunk); err != nil { return err }
		wroteChunk = true
	}
	
	// チャンクが1つもない場合（空ファイル）でも writer を呼び、出力先の作成を保証する
	if !wroteChunk {
		if err := writer(name, []byte{}); err != nil { return err }
	}
	
	return nil
}
