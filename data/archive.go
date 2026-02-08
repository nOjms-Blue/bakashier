package data

import (
	"bakashier/utils"

	"bytes"
	"encoding/binary"
	"errors"
	"os"
)

// アーカイブ内の「名前」と「データ」の生バイトを保持する。
// 実体は暗号化・圧縮された内容であり、FromArchiveData で復号・展開する。
type ArchiveData struct {
	Name []byte
	Data []byte
}

// fileName の .bks ファイルを読み、ヘッダー検証と CRC32 チェック後に d に格納する。
// フォーマット: "BKS" + version(2) + nameLen(4) + dataLen(8) + name + data + CRC32(4)
func (d *ArchiveData) Import(fileName string) error {
	content, err := os.ReadFile(fileName)
	if err != nil { return err }
	if len(content) < 17 { return errors.New("file is too short") }
	if content[0] != byte('B') || content[1] != byte('K') || content[2] != byte('S') { return errors.New("file is not a valid archived file") }
	if binary.BigEndian.Uint16(content[3:5]) != 1 { return errors.New("file is not a valid archived file") }
	
	archived_name_len := binary.BigEndian.Uint32(content[5:9])
	archived_data_len := binary.BigEndian.Uint64(content[9:17])
	name_end := 17 + uint64(archived_name_len)
	data_end := name_end + archived_data_len
	d.Name = content[17:name_end]
	d.Data = content[name_end:data_end]
	
	import_hash := content[data_end:]
	calc_hash := utils.CRC32HashBytes(content[17:data_end])
	if !bytes.Equal(import_hash, calc_hash) {
		return errors.New("file is not a valid archived file (hash mismatch)")
	}
	return nil
}

// d の内容を .bks 形式で fileName に書き出す。name/data の後に CRC32 を付加する。
func (d ArchiveData) Export(fileName string) error {
	var content []byte
	var version_bin = make([]byte, 2)
	var archived_name_len_bin  = make([]byte, 4)
	var archived_data_len_bin  = make([]byte, 8)
	var name_and_data []byte
	
	binary.BigEndian.PutUint16(version_bin, 1)
	binary.BigEndian.PutUint32(archived_name_len_bin, uint32(len(d.Name)))
	binary.BigEndian.PutUint64(archived_data_len_bin, uint64(len(d.Data)))
	
	name_and_data = append(name_and_data, d.Name...)
	name_and_data = append(name_and_data, d.Data...)
	calc_hash := utils.CRC32HashBytes(name_and_data)
	
	content = append(content, byte('B'))
	content = append(content, byte('K'))
	content = append(content, byte('S'))
	content = append(content, version_bin...)
	content = append(content, archived_name_len_bin...)
	content = append(content, archived_data_len_bin...)
	content = append(content, name_and_data...)
	content = append(content, calc_hash...)
	
	return os.WriteFile(fileName, content, 0644)
}
