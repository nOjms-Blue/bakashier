package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	legacyPBKDF2Iterations = 4096
	pbkdf2Iterations       = 600000
	passwordCipherSaltSize = 16
)

var passwordCipherMagic = [8]byte{'B', 'K', 'S', 'E', 'N', 'C', '0', '2'}

// パスワードから PBKDF2 で鍵を導出し、AES-GCM でバイト列を暗号化する。
// 戻り値は magic(8) + iterations(4) + salt(16) + nonce + ciphertext の形式。
func EncryptBytesWithPassword(plainData []byte, password string) ([]byte, error) {
	salt := make([]byte, passwordCipherSaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	cipherText := gcm.Seal(nil, nonce, plainData, nil)

	// magic | iterations | salt | nonce | ciphertext
	result := make([]byte, 0, len(passwordCipherMagic)+4+len(salt)+len(nonce)+len(cipherText))
	result = append(result, passwordCipherMagic[:]...)
	iterationBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(iterationBytes, pbkdf2Iterations)
	result = append(result, iterationBytes...)
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, cipherText...)

	return result, nil
}

// EncryptBytesWithPassword で暗号化したデータを、同じパスワードで復号する。
// 旧形式の salt(16) + nonce + ciphertext も読み取り可能。
func DecryptBytesWithPassword(cipherData []byte, password string) ([]byte, error) {
	saltOffset := 0
	iterations := legacyPBKDF2Iterations
	if len(cipherData) >= len(passwordCipherMagic) &&
		string(cipherData[:len(passwordCipherMagic)]) == string(passwordCipherMagic[:]) {
		headerSize := len(passwordCipherMagic) + 4
		if len(cipherData) < headerSize+passwordCipherSaltSize {
			return nil, errors.New("ciphertext too short (incomplete password cipher header)")
		}
		iterations = int(binary.BigEndian.Uint32(cipherData[len(passwordCipherMagic):headerSize]))
		if iterations != pbkdf2Iterations {
			return nil, fmt.Errorf("unsupported PBKDF2 iteration count: %d", iterations)
		}
		saltOffset = headerSize
	}

	if len(cipherData) < saltOffset+passwordCipherSaltSize {
		return nil, errors.New("ciphertext too short (no salt)")
	}
	salt := cipherData[saltOffset : saltOffset+passwordCipherSaltSize]
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonceOffset := saltOffset + passwordCipherSaltSize
	if len(cipherData) < nonceOffset+nonceSize {
		return nil, errors.New("ciphertext too short (no nonce)")
	}
	nonce := cipherData[nonceOffset : nonceOffset+nonceSize]
	cipherText := cipherData[nonceOffset+nonceSize:]

	plainData, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return plainData, nil
}
