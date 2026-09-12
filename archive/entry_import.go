package archive

import (
	"encoding/binary"
	"errors"
	"io"
	"time"
)

// バイナリ列をパースし、DirectoryEntry に変換していく。
func ImportDirectoryEntries(reader io.Reader, writer func(entry DirectoryEntry) error) error {
	for {
		// Type のバイトを取得
		typ := [1]byte{0}
		n, err := io.ReadFull(reader, typ[:])
		if n == 0 || err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Type の検証
		entryType := Unknown
		if typ[0] == byte(File) {
			entryType = File
		}
		if typ[0] == byte(Directory) {
			entryType = Directory
		}

		// RealName のバイト長を取得
		realNameBytesLengthBytes := [4]byte{0, 0, 0, 0}
		n, err = io.ReadFull(reader, realNameBytesLengthBytes[:])
		if err != nil {
			return err
		}
		realNameBytesLength := binary.BigEndian.Uint32(realNameBytesLengthBytes[:])
		if realNameBytesLength > MAX_NAME_SIZE {
			return errors.New("real name length is out of range")
		}

		// HideName のバイト長を取得
		hideNameBytesLengthBytes := [4]byte{0, 0, 0, 0}
		n, err = io.ReadFull(reader, hideNameBytesLengthBytes[:])
		if err != nil {
			return err
		}
		hideNameBytesLength := binary.BigEndian.Uint32(hideNameBytesLengthBytes[:])
		if hideNameBytesLength > MAX_NAME_SIZE {
			return errors.New("hide name length is out of range")
		}

		// RealName のバイト列を取得
		realNameBytes := make([]byte, realNameBytesLength)
		_, err = io.ReadFull(reader, realNameBytes)
		if err != nil {
			return err
		}

		// HideName のバイト列を取得
		hideNameBytes := make([]byte, hideNameBytesLength)
		_, err = io.ReadFull(reader, hideNameBytes)
		if err != nil {
			return err
		}

		// Size を取得
		sizeBytes := [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
		_, err = io.ReadFull(reader, sizeBytes[:])
		if err != nil {
			return err
		}
		size := binary.BigEndian.Uint64(sizeBytes[:])

		// ModTimeNano を取得
		modTimeNanoBytes := [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
		_, err = io.ReadFull(reader, modTimeNanoBytes[:])
		if err != nil {
			return err
		}
		modTimeNano := int64(binary.BigEndian.Uint64(modTimeNanoBytes[:]))

		// writer で返す
		err = writer(DirectoryEntry{
			Type:     entryType,
			RealName: string(realNameBytes),
			HideName: string(hideNameBytes),
			Size:     size,
			ModTime:  time.Unix(0, modTimeNano),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
