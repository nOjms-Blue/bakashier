package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func encryptLegacyForTest(t *testing.T, plainData []byte, password string) []byte {
	t.Helper()
	salt := bytes.Repeat([]byte{1}, passwordCipherSaltSize)
	key := pbkdf2.Key([]byte(password), salt, legacyPBKDF2Iterations, 32, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := bytes.Repeat([]byte{2}, gcm.NonceSize())
	result := append([]byte{}, salt...)
	result = append(result, nonce...)
	return gcm.Seal(result, nonce, plainData, nil)
}

func TestPasswordEncryptionUsesVersionedStrongKDF(t *testing.T) {
	plainData := []byte("secret data")
	encrypted, err := EncryptBytesWithPassword(plainData, "password")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(encrypted, passwordCipherMagic[:]) {
		t.Fatal("encrypted data does not contain the versioned password cipher header")
	}
	iterations := binary.BigEndian.Uint32(encrypted[len(passwordCipherMagic) : len(passwordCipherMagic)+4])
	if iterations != pbkdf2Iterations {
		t.Fatalf("PBKDF2 iterations = %d, want %d", iterations, pbkdf2Iterations)
	}

	decrypted, err := DecryptBytesWithPassword(encrypted, "password")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plainData) {
		t.Fatalf("decrypted data = %q, want %q", decrypted, plainData)
	}
}

func TestPasswordDecryptionSupportsLegacyCiphertext(t *testing.T) {
	plainData := []byte("legacy secret")
	encrypted := encryptLegacyForTest(t, plainData, "password")

	decrypted, err := DecryptBytesWithPassword(encrypted, "password")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plainData) {
		t.Fatalf("decrypted data = %q, want %q", decrypted, plainData)
	}
}

func TestPasswordDecryptionRejectsUnsupportedIterationCount(t *testing.T) {
	encrypted, err := EncryptBytesWithPassword([]byte("secret"), "password")
	if err != nil {
		t.Fatal(err)
	}
	binary.BigEndian.PutUint32(encrypted[len(passwordCipherMagic):len(passwordCipherMagic)+4], 1)

	if _, err := DecryptBytesWithPassword(encrypted, "password"); err == nil {
		t.Fatal("expected unsupported iteration count to be rejected")
	}
}
