package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptBytesWithPassword: パスワードでbyte配列を暗号化する
func EncryptBytesWithPassword(plainData []byte, password string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	key := pbkdf2.Key([]byte(password), salt, 4096, 32, sha256.New)

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

	// salt | nonce | ciphertext
	result := append(salt, nonce...)
	result = append(result, cipherText...)

	return result, nil
}

// DecryptBytesWithPassword: パスワードでbyte配列を復号化する
func DecryptBytesWithPassword(cipherData []byte, password string) ([]byte, error) {
	if len(cipherData) < 16 {
		return nil, errors.New("ciphertext too short (no salt)")
	}
	salt := cipherData[:16]
	key := pbkdf2.Key([]byte(password), salt, 4096, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherData) < 16+nonceSize {
		return nil, errors.New("ciphertext too short (no nonce)")
	}
	nonce := cipherData[16 : 16+nonceSize]
	cipherText := cipherData[16+nonceSize:]

	plainData, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return plainData, nil
}