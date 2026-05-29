package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
)

// データの CRC32(IEEE) を計算し、4バイト（BigEndian）で返す。
func CRC32HashBytes(data []byte) []byte {
	hash := crc32.ChecksumIEEE(data)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, hash)
	return buf
}

// ファイルのSHA256のハッシュ値を求める
func CalcSHA256FromFile(file string) ([]byte, error) {
	fp, err := os.Open(file)
	if err != nil {
		return []byte{}, err
	}
	defer fp.Close()
	
	hasher := sha256.New()
	if _, err := io.Copy(hasher, fp); err != nil {
		return []byte{}, err
	}
	
	return hasher.Sum(nil), nil
}
