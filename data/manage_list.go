package data

import (
	"bakashier/utils"
	"bytes"
	"encoding/binary"
	"errors"
)


type ManageListRow struct {
	Hash    []byte
	OrgPath string
	BksFile string
}

type ManageList struct {
	file string
	list []ManageListRow
}

func (m *ManageList) Load(file string, password string) error {
	_, content, err := ImportAll(file, password)
	if err != nil { return err }
	
	for {
		if len(content) < utils.SHA256_BYTES { break }
		hash := content[:utils.SHA256_BYTES]
		
		content = content[utils.SHA256_BYTES:]
		
		if len(content) < 4 { break }
		length := binary.BigEndian.Uint32(content[:4])
		
		if uint64(len(content)) < 4 + uint64(length) { break }
		orgPath := string(content[4:4+length])
		
		content = content[4+length:]
		
		if len(content) < 2 { break }
		length = uint32(binary.BigEndian.Uint16(content[:2]))
		
		if uint64(len(content)) < 2 + uint64(length) { break }
		bksFile := string(content[2:2+length])
		
		m.list = append(m.list, ManageListRow{
			Hash: hash,
			OrgPath: orgPath,
			BksFile: bksFile,
		})
	}
	
	m.file = file
	return nil
}

func (m ManageList) Save(password string, chunkSize uint64) error {
	var buf bytes.Buffer
	
	for _, row := range m.list {
		if len(row.Hash) != utils.SHA256_BYTES {
			return errors.New("manage list row format error: row.Hash is not SHA256 hash bytes")
		}
		_, err := buf.Write(row.Hash)
		if err != nil { return err }
		
		orgPathBytes := []byte(row.OrgPath)
		orgPathLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(orgPathLenBytes, uint32(len(orgPathBytes)))
		_, err = buf.Write(orgPathLenBytes)
		if err != nil { return err }
		_, err = buf.Write(orgPathBytes)
		if err != nil { return err }
		
		bksFileBytes := []byte(row.BksFile)
		bksFileLenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(bksFileLenBytes, uint16(len(bksFileBytes)))
		_, err = buf.Write(bksFileLenBytes)
		if err != nil { return err }
		_, err = buf.Write(bksFileBytes)
		if err != nil { return err }
	}
	
	return ExportAllV2(m.file, buf.Bytes(), password, chunkSize)
}

func (m ManageList) Get() []ManageListRow {
	copied := make([]ManageListRow, len(m.list))
	copy(copied, m.list)
	return copied
}

func (m *ManageList) Set(list []ManageListRow) {
	m.list = make([]ManageListRow, len(list))
	copy(m.list, list)
}

func (m *ManageList) Add(row ...ManageListRow) {
	m.list = append(m.list, row...)
}

func (m *ManageList) DeleteFromOrgPath(orgPath string) {
	var index = 0
	
	for _, row := range m.list {
		if row.OrgPath != orgPath {
			m.list[index] = row
			index = index + 1
		}
	}
	m.list = m.list[:index]
}

func (m *ManageList) DeleteFromBksFile(bksFile string) {
	var index = 0
	
	for _, row := range m.list {
		if row.BksFile != bksFile {
			m.list[index] = row
			index = index + 1
		}
	}
	m.list = m.list[:index]
}
