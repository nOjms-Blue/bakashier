package data

import (
	"bakashier/utils"
	"bytes"
	"errors"
	"os"
	"path/filepath"
)

func Verify(file string, password string, hash []byte) (bool, error) {
	if len(hash) != utils.SHA256_BYTES {
		return false, errors.New("not SHA256 hash bytes")
	}
	
	version, err := CheckVerBks(file)
	if err != nil {
		return false, err
	}
	
	tempDir, err := os.MkdirTemp("", "bakashier-*")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tempDir)
	
	restored := ""
	switch version {
	case 1:
		restored, err = ImportStreamArchive(file, tempDir, password)
		if err == ERR_HASH_MISMATCH {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	case 2:
		restored = filepath.Join(tempDir, "restore.dat")
		
		err := ImportStreamArchiveV2(file, restored, password)
		if err == ERR_HASH_MISMATCH {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}
	
	newHash, err := utils.CalcSHA256FromFile(restored)
	if err != nil {
		return false, err
	}
	
	return bytes.Equal(hash, newHash), nil
}
