package data

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	
	"bakashier/utils"
)


const chunkSize = 16 * 1024 * 1024 // 16MB

func ExportStreamArchive(srcFile string, destFile string, fileName string, password string) error {
	// ソースファイルを開く
	src, err := os.Open(srcFile)
	if err != nil { return err }
	defer src.Close()
	
	// 書き出し先ファイルを開く
	dest, err := os.Create(destFile)
	if err != nil { return err }
	defer dest.Close()
	
	// ファイル名を圧縮・暗号化
	nameBytes := []byte(fileName)
	compressedName, err := utils.CompressBytes(nameBytes)
	if err != nil { return err }
	encryptedName, err := utils.EncryptBytesWithPassword(compressedName, password)
	if err != nil { return err }
	
	// ヘッダを書き込む
	var versionBin = make([]byte, 2)
	var nameLenBin = make([]byte, 4)
	binary.BigEndian.PutUint16(versionBin, 1)
	binary.BigEndian.PutUint32(nameLenBin, uint32(len(encryptedName)))
	dest.Write([]byte("BKS"))
	dest.Write(versionBin)
	dest.Write(nameLenBin)
	dest.Write(encryptedName)
	dest.Write(utils.CRC32HashBytes(encryptedName))
	
	for {
		chunk := make([]byte, chunkSize)
		n, err := src.Read(chunk)
		if err == io.EOF || n == 0 { break }
		if err != nil { return err }
		
		// CRC32 ハッシュを計算
		chunkCRC := utils.CRC32HashBytes(chunk)
		
		// 圧縮 → 暗号化
		chunkCompressed, err := utils.CompressBytes(chunk)
		if err != nil { return err }
		chunkEncrypted, err := utils.EncryptBytesWithPassword(chunkCompressed, password)
		if err != nil { return err }
		
		// チャンク長を書き込む
		chunkLenBin := make([]byte, 8)
		binary.BigEndian.PutUint64(chunkLenBin, uint64(len(chunkEncrypted)))
		
		// チャンクを書き込む
		dest.Write(chunkLenBin)
		dest.Write(chunkEncrypted)
		dest.Write(chunkCRC)
	}
	
	return nil
}

func ImportStreamArchive(archiveFile string, destDirectory string, password string) (error, string) {
	// アーカイブファイルを開く
	archive, err := os.Open(archiveFile)
	if err != nil { return err, "" }
	defer archive.Close()
	
	// ヘッダを読み込む
	header := make([]byte, 9)
	_, err = archive.Read(header)
	if err != nil { return err, "" }
	if header[0] != byte('B') || header[1] != byte('K') || header[2] != byte('S') {
		return errors.New("file is not a valid archived file"), ""
	}
	if binary.BigEndian.Uint16(header[3:5]) != 1 {
		return errors.New("unsupported version number"), ""
	}
	
	// 名前情報の取得
	nameLen := binary.BigEndian.Uint32(header[5:9])
	nameBytes := make([]byte, nameLen)
	nameHash := make([]byte, 4)
	_, err = archive.Read(nameBytes)
	if err != nil { return err, "" }
	decryptedName, err := utils.DecryptBytesWithPassword(nameBytes, password)
	if err != nil { return err, "" }
	decompressedName, err := utils.DecompressBytes(decryptedName)
	if err != nil { return err, "" }
	name := string(decompressedName)
	_, err = archive.Read(nameHash)
	if err != nil { return err, "" }
	if !bytes.Equal(nameHash, utils.CRC32HashBytes(nameBytes)) {
		return errors.New("name hash mismatch"), ""
	}
	
	// 書き出し先ファイルを開く
	destFile := filepath.Join(destDirectory, name)
	dest, err := os.Create(destFile)
	if err != nil { return err, "" }
	defer dest.Close()
	
	// チャンクを読み込む
	for {
		// チャンク長を読み込む
		chunkLenBin := make([]byte, 8)
		_, err = archive.Read(chunkLenBin)
		if err == io.EOF { break }
		if err != nil { return err, "" }
		chunkLen := binary.BigEndian.Uint64(chunkLenBin)
		
		// チャンクを読み込む
		chunk := make([]byte, chunkLen)
		_, err = archive.Read(chunk)
		if err != nil { return err, "" }
		
		// CRC32 ハッシュを読み込む
		chunkCRC := make([]byte, 4)
		_, err = archive.Read(chunkCRC)
		if err != nil { return err, "" }
		
		// チャンクを復号・展開
		chunkDecrypted, err := utils.DecryptBytesWithPassword(chunk, password)
		if err != nil { return err, "" }
		chunkDecompressed, err := utils.DecompressBytes(chunkDecrypted)
		if err != nil { return err, "" }
		
		// チャンクを書き出す
		dest.Write(chunkDecompressed)
		
		// CRC32 ハッシュを検証
		if !bytes.Equal(chunkCRC, utils.CRC32HashBytes(chunkDecompressed)) {
			return errors.New("chunk CRC32 hash mismatch"), ""
		}
	}
	
	return nil, destFile
}
