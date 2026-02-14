package data

import (
	"encoding/binary"
	"errors"
	"os"
)


type ArchiveEntry struct {
	Data []byte
	Hash []byte
}

// アーカイブ内の「名前」と「データ」の生バイトを保持する。
// 実体は暗号化・圧縮された内容であり、FromArchiveData で復号・展開する。
type ArchiveData struct {
	Name ArchiveEntry
	Data []ArchiveEntry
}

// fileName の .bks ファイルを読み、ヘッダー検証と CRC32 チェック後に d に格納する。
// フォーマット: "BKS" + version(2) + nameLen(4) + name + CRC32(4) + dataLen(8) + data + CRC32(4) + dataLen(8) + data + CRC32(4) + ...
func (d *ArchiveData) Import(fileName string) error {
	content, err := os.ReadFile(fileName)
	if err != nil { return err }
	if len(content) < 9 { return errors.New("file is too short") }
	if content[0] != byte('B') || content[1] != byte('K') || content[2] != byte('S') { return errors.New("file is not a valid archived file") }
	if binary.BigEndian.Uint16(content[3:5]) != 1 { return errors.New("unsupported version number") }
	
	archived_name_len := binary.BigEndian.Uint32(content[5:9])
	name_end := 9 + uint32(archived_name_len)
	d.Name = ArchiveEntry{
		Data: content[9:name_end],
		Hash: content[name_end:name_end+4],
	}
	
	data_start := uint64(name_end) + 4
	for data_start < uint64(len(content)) {
		data_len := binary.BigEndian.Uint64(content[data_start:data_start+8])
		data_end := data_start + 8 + uint64(data_len)
		d.Data = append(d.Data, ArchiveEntry{
			Data: content[data_start+8:data_end],
			Hash: content[data_end:data_end+4],
		})
		data_start = data_end + 4
	}
	return nil
}

// d の内容を .bks 形式で fileName に書き出す。name/data の後に CRC32 を付加する。
func (d ArchiveData) Export(fileName string) error {
	var content []byte
	var version_bin = make([]byte, 2)
	var archived_name_len_bin  = make([]byte, 4)
	var archived_data_len_bin  = make([]byte, 8)
	binary.BigEndian.PutUint16(version_bin, 1)
	
	content = append(content, byte('B'))
	content = append(content, byte('K'))
	content = append(content, byte('S'))
	content = append(content, version_bin...)
	
	// 名前
	binary.BigEndian.PutUint32(archived_name_len_bin, uint32(len(d.Name.Data)))
	content = append(content, archived_name_len_bin...)
	content = append(content, d.Name.Data...)
	content = append(content, d.Name.Hash...)
	
	// データ
	for _, data := range d.Data {
		binary.BigEndian.PutUint64(archived_data_len_bin, uint64(len(data.Data)))
		content = append(content, archived_data_len_bin...)
		content = append(content, data.Data...)
		content = append(content, data.Hash...)
	}
	
	return os.WriteFile(fileName, content, 0644)
}
