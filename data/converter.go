package data

import (
	"bakashier/utils"

	"errors"
)

// ファイル名・ファイル内容・パスワードを受け取り、ArchiveData に変換する。
func ToArchiveData(filename string, content []byte, password string) (ArchiveData, error) {
	if password == "" {
		return ArchiveData{}, errors.New("password is required")
	}
	
	nameBytes := []byte(filename)
	compressedName, err := utils.CompressBytes(nameBytes)
	if err != nil { return ArchiveData{}, err }
	encryptedName, err := utils.EncryptBytesWithPassword(compressedName, password)
	if err != nil { return ArchiveData{}, err }
	compressedContent, err := utils.CompressBytes(content)
	if err != nil { return ArchiveData{}, err }
	encryptedContent, err := utils.EncryptBytesWithPassword(compressedContent, password)
	if err != nil { return ArchiveData{}, err }
	
	return ArchiveData{
		Name: encryptedName,
		Data: encryptedContent,
	}, nil
}

// ArchiveData・パスワードを受け取り、ファイル名とファイル内容に戻す。
func FromArchiveData(archive ArchiveData, password string) (filename string, content []byte, err error) {
	if password == "" {
		return "", nil, errors.New("password is required")
	}
	
	decryptedName, err := utils.DecryptBytesWithPassword(archive.Name, password)
	if err != nil { return "", nil, err }
	nameBytes, err := utils.DecompressBytes(decryptedName)
	if err != nil { return "", nil, err }
	decryptedContent, err := utils.DecryptBytesWithPassword(archive.Data, password)
	if err != nil { return "", nil, err }
	content, err = utils.DecompressBytes(decryptedContent)
	if err != nil { return "", nil, err }
	return string(nameBytes), content, nil
}
