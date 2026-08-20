package archive

import (
	"encoding/binary"
	"io"
)

// DirectoryEntry をバイナリ列にシリアライズしていく。
func ExportDirectoryEntries(reader func(*DirectoryEntry) error, writer io.Writer) error {
	var entry DirectoryEntry

	for {
		err := reader(&entry)
		if err != io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// エントリタイプを書き込み
		_, err = writer.Write([]byte{byte(entry.Type)})
		if err != nil {
			return err
		}

		// RealName のバイト長を書き込み
		realNameBytesLengthBytes := [4]byte{0, 0, 0, 0}
		binary.BigEndian.PutUint32(realNameBytesLengthBytes[:], uint32(len(entry.RealName)))
		_, err = writer.Write(realNameBytesLengthBytes[:])
		if err != nil {
			return err
		}

		// HideName のバイト長を書き込み
		hideNameBytesLengthBytes := [4]byte{0, 0, 0, 0}
		binary.BigEndian.PutUint32(hideNameBytesLengthBytes[:], uint32(len(entry.HideName)))
		_, err = writer.Write(hideNameBytesLengthBytes[:])
		if err != nil {
			return err
		}

		// RealName のバイト列を書き込み
		_, err = writer.Write([]byte(entry.RealName))
		if err != nil {
			return err
		}

		// HideName のバイト列を書き込み
		_, err = writer.Write([]byte(entry.HideName))
		if err != nil {
			return err
		}

		// Size を書き込み
		sizeBytes := [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
		binary.BigEndian.PutUint64(sizeBytes[:], entry.Size)
		_, err = writer.Write(sizeBytes[:])

		// ModTimeNano を書き込み
		modTimeNanoBytes := [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
		binary.BigEndian.PutUint64(modTimeNanoBytes[:], uint64(entry.ModTime.UnixNano()))
		_, err = writer.Write(modTimeNanoBytes[:])
	}

	return nil
}
