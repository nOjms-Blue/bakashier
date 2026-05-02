package data

import (
	"bakashier/utils"
	"bytes"
	"encoding/binary"
	"errors"
)


func ExportBks(name string, reader func(length uint64) ([]byte, error), writer func(data []byte) error, password string, chunkSize uint64) error {
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
	writer([]byte("BKS"))
	writer(versionBytes)
	writer(nameLenBytes)
	writer(nameBytes)
	writer(nameCRC)
	
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
		
		writer(chunkLenBytes)
		writer(chunk)
		writer(chunkCRC)
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
	
	// ヘッダの先頭部分の検証
	if header[0] != byte('B') || header[1] != byte('K') || header[2] != byte('S') {
		return errors.New("file is not a valid archived file")
	}
	if binary.BigEndian.Uint16(header[3:5]) != 1 {
		return errors.New("unsupported version number")
	}
	
	// 名前情報の取得
	nameLen := binary.BigEndian.Uint32(header[5:9])
	nameBytes, err := reader(uint64(nameLen))
	if err != nil { return err }
	nameHash, err := reader(4)
	if err != nil { return err }
	nameBytes, err = importProcess(nameBytes, nameHash)
	if err != nil { return err }
	name := string(nameBytes)

	chunkLenBytes := []byte{}
	chunk := []byte{}
	chunkCRC := []byte{}
	for {
		chunkLenBytes, err = reader(8)
		if err != nil { return err }
		if len(chunkLenBytes) == 0 { break }
		chunkLen := binary.BigEndian.Uint64(chunkLenBytes)
		
		// チャンクを読み込む
		chunk, err = reader(chunkLen)
		if err != nil { return err }
		
		// CRC32 ハッシュを読み込む
		chunkCRC, err = reader(4)
		if err != nil { return err }
		
		// チャンクを復号・展開
		chunk, err = importProcess(chunk, chunkCRC)
		if err != nil { return err }
		
		writer(name, chunk)
	}
	
	return nil
}
