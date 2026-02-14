package data

import (
	"bakashier/utils"
	"bytes"
	
	"errors"
)


// ファイル名・ファイル内容・パスワードを受け取り、圧縮・暗号化した ArchiveData に変換する。
func ToArchiveData(filename string, content []byte, password string) (ArchiveData, error) {
	if password == "" {
		return ArchiveData{}, errors.New("password is required")
	}
	
	// 名前
	nameBytes := []byte(filename)
	nameHash := utils.CRC32HashBytes(nameBytes)
	compressedName, err := utils.CompressBytes(nameBytes)
	if err != nil { return ArchiveData{}, err }
	encryptedName, err := utils.EncryptBytesWithPassword(compressedName, password)
	if err != nil { return ArchiveData{}, err }
	
	// データ
	contentHash := utils.CRC32HashBytes(content)
	compressedContent, err := utils.CompressBytes(content)
	if err != nil { return ArchiveData{}, err }
	encryptedContent, err := utils.EncryptBytesWithPassword(compressedContent, password)
	if err != nil { return ArchiveData{}, err }
	
	return ArchiveData{
		Name: ArchiveEntry{
			Data: encryptedName,
			Hash: nameHash,
		},
		Data: []ArchiveEntry{
			{
				Data: encryptedContent,
				Hash: contentHash,
			},
		},
	}, nil
}

// ArchiveData とパスワードを受け取り、復号・展開してファイル名とファイル内容に戻す。
func FromArchiveData(archive ArchiveData, password string) (filename string, content []byte, err error) {
	if password == "" {
		return "", nil, errors.New("password is required")
	}
	
	// 名前
	decryptedName, err := utils.DecryptBytesWithPassword(archive.Name.Data, password)
	if err != nil { return "", nil, err }
	nameBytes, err := utils.DecompressBytes(decryptedName)
	if err != nil { return "", nil, err }
	nameHash := utils.CRC32HashBytes(nameBytes)
	if !bytes.Equal(nameHash, archive.Name.Hash) {
		return "", nil, errors.New("file is not a valid archived file (hash mismatch)")
	}
	
	content = []byte{}
	for _, data := range archive.Data {
		decryptedContent, err := utils.DecryptBytesWithPassword(data.Data, password)
		if err != nil { return "", nil, err }
		content, err = utils.DecompressBytes(decryptedContent)
		if err != nil { return "", nil, err }
		contentHash := utils.CRC32HashBytes(content)
		if !bytes.Equal(contentHash, data.Hash) {
			return "", nil, errors.New("file is not a valid archived file (hash mismatch)")
		}
		
		content = append(content, data.Hash...)
	}
	
	return string(nameBytes), content, nil
}
