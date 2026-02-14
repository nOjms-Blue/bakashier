package data

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"
)

// エントリがディレクトリかファイルかを表す。
type DirectoryEntryType byte
const (
	Unknown   DirectoryEntryType = 'U'
	Directory DirectoryEntryType = 'D'
	File      DirectoryEntryType = 'F'
)

// 1つのファイルまたはディレクトリの実名・隠し名・サイズ・更新日時を保持する。
type DirectoryEntry struct {
	Type     DirectoryEntryType
	RealName string
	HideName string
	Size     uint64
	ModTime  time.Time
}

// 1エントリの固定長ヘッダー: Type(1) + RealNameLen(4) + HideNameLen(4) + Size(8) + ModTime(8) = 25
const dirEntryHeaderSize = 1 + 4 + 4 + 8 + 8

// バイナリ列をパースし、DirectoryEntry のスライスに変換する。
func ImportDirectoryEntries(content []byte) ([]DirectoryEntry, error) {
	var entries []DirectoryEntry
	r := bytes.NewReader(content)
	for r.Len() >= dirEntryHeaderSize {
		var typ DirectoryEntryType
		if err := binary.Read(r, binary.BigEndian, &typ); err != nil {
			return nil, err
		}
		var realNameLen, hideNameLen uint32
		if err := binary.Read(r, binary.BigEndian, &realNameLen); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.BigEndian, &hideNameLen); err != nil {
			return nil, err
		}
		remaining := int64(r.Len())
		if int64(realNameLen)+int64(hideNameLen)+16 > remaining {
			return nil, errors.New("directory entry: invalid or truncated entry")
		}
		realName := make([]byte, realNameLen)
		if _, err := r.Read(realName); err != nil {
			return nil, err
		}
		hideName := make([]byte, hideNameLen)
		if _, err := r.Read(hideName); err != nil {
			return nil, err
		}
		var size uint64
		if err := binary.Read(r, binary.BigEndian, &size); err != nil {
			return nil, err
		}
		var modTimeNano int64
		if err := binary.Read(r, binary.BigEndian, &modTimeNano); err != nil {
			return nil, err
		}
		entries = append(entries, DirectoryEntry{
			Type:     typ,
			RealName: string(realName),
			HideName: string(hideName),
			Size:     size,
			ModTime:  time.Unix(0, modTimeNano),
		})
	}
	return entries, nil
}

// DirectoryEntry のスライスをバイナリ列にシリアライズする。
func ExportDirectoryEntries(entries []DirectoryEntry) ([]byte, error) {
	var buf bytes.Buffer
	for _, e := range entries {
		if err := binary.Write(&buf, binary.BigEndian, e.Type); err != nil {
			return nil, err
		}
		realName := []byte(e.RealName)
		hideName := []byte(e.HideName)
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(realName))); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(hideName))); err != nil {
			return nil, err
		}
		if _, err := buf.Write(realName); err != nil {
			return nil, err
		}
		if _, err := buf.Write(hideName); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.Size); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.ModTime.UnixNano()); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
