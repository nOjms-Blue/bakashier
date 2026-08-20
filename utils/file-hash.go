package utils

import (
	"encoding/binary"
	"hash/crc32"
)

// データの CRC32(IEEE) を計算し、4バイト（BigEndian）で返す。
func CRC32HashBytes(data []byte) []byte {
	hash := crc32.ChecksumIEEE(data)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, hash)
	return buf
}
