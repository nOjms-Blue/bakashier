package utils

import (
	"hash/crc32"
	"encoding/binary"
)

// CRC32で[]byteをハッシュ化し、そのハッシュ値を[]byteで返す
func CRC32HashBytes(data []byte) []byte {
	hash := crc32.ChecksumIEEE(data)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, hash)
	return buf
}
