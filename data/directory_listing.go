package data

import (
	"encoding/binary"
	"io"

	"os"
)

// DirectoryEntry はディレクトリ直下の1エントリ（ファイルまたはサブディレクトリ）の情報。
// ディレクトリの場合は Size は 0、ModTime は利用しない。
type DirectoryEntry struct {
	Name    string
	Size    int64
	ModTime int64
}

// BuildDirectoryListingData は dirPath の直下のファイル・サブディレクトリ一覧を
// ArchiveData.Data に格納するためのバイト列に直列化する。
// 形式: [エントリ数: 4byte BE][エントリ...]
// 各エントリ: [名前長: 2byte BE][名前][サイズ: 8byte BE][最終更新日時(Unix): 8byte BE]
func BuildDirectoryListingData(dirPath string) ([]byte, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	var buf []byte
	numBin := make([]byte, 4)
	binary.BigEndian.PutUint32(numBin, uint32(len(entries)))
	buf = append(buf, numBin...)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		name := e.Name()
		size := int64(0)
		modTime := int64(0)
		if !e.IsDir() {
			size = info.Size()
			modTime = info.ModTime().Unix()
		}
		nameBytes := []byte(name)
		nameLenBin := make([]byte, 2)
		binary.BigEndian.PutUint16(nameLenBin, uint16(len(nameBytes)))
		buf = append(buf, nameLenBin...)
		buf = append(buf, nameBytes...)
		sizeBin := make([]byte, 8)
		modBin := make([]byte, 8)
		binary.BigEndian.PutUint64(sizeBin, uint64(size))
		binary.BigEndian.PutUint64(modBin, uint64(modTime))
		buf = append(buf, sizeBin...)
		buf = append(buf, modBin...)
	}
	return buf, nil
}

// ParseDirectoryListingData は BuildDirectoryListingData で直列化したバイト列をパースする。
func ParseDirectoryListingData(data []byte) ([]DirectoryEntry, error) {
	if len(data) < 4 {
		return nil, nil
	}
	n := binary.BigEndian.Uint32(data[:4])
	out := make([]DirectoryEntry, 0, n)
	off := 4
	for i := uint32(0); i < n && off < len(data); i++ {
		if off+2 > len(data) {
			break
		}
		nameLen := binary.BigEndian.Uint16(data[off : off+2])
		off += 2
		if off+int(nameLen)+8+8 > len(data) {
			break
		}
		name := string(data[off : off+int(nameLen)])
		off += int(nameLen)
		size := int64(binary.BigEndian.Uint64(data[off : off+8]))
		off += 8
		modTime := int64(binary.BigEndian.Uint64(data[off : off+8]))
		off += 8
		out = append(out, DirectoryEntry{Name: name, Size: size, ModTime: modTime})
	}
	return out, nil
}

// ReadDirectoryListingData は io.Reader からディレクトリ一覧を読み取る。
func ReadDirectoryListingData(r io.Reader) ([]DirectoryEntry, error) {
	var numBin [4]byte
	if _, err := io.ReadFull(r, numBin[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(numBin[:])
	out := make([]DirectoryEntry, 0, n)
	for i := uint32(0); i < n; i++ {
		var nameLenBin [2]byte
		if _, err := io.ReadFull(r, nameLenBin[:]); err != nil {
			return nil, err
		}
		nameLen := binary.BigEndian.Uint16(nameLenBin[:])
		nameBytes := make([]byte, nameLen)
		if _, err := io.ReadFull(r, nameBytes); err != nil {
			return nil, err
		}
		var sizeBin [8]byte
		if _, err := io.ReadFull(r, sizeBin[:]); err != nil {
			return nil, err
		}
		var modBin [8]byte
		if _, err := io.ReadFull(r, modBin[:]); err != nil {
			return nil, err
		}
		out = append(out, DirectoryEntry{
			Name:    string(nameBytes),
			Size:    int64(binary.BigEndian.Uint64(sizeBin[:])),
			ModTime: int64(binary.BigEndian.Uint64(modBin[:])),
		})
	}
	return out, nil
}
