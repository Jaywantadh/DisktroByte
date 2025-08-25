package encryptor

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/scrypt"
)

const (
	saltSize   = 16
	nonceSize  = chacha20poly1305.NonceSize
	keySize    = chacha20poly1305.KeySize
	scryptN    = 32768
	scryptR    = 8
	scryptP    = 1
)

// Encryptor defines the interface for encryption and decryption operations.
type Encryptor interface {
	Encrypt(plaintext []byte, password string) ([]byte, error)
	Decrypt(ciphertext []byte, password string) ([]byte, error)
}

// chaCha20Poly1305Encryptor implements the Encryptor interface using ChaCha20-Poly1305.
type chaCha20Poly1305Encryptor struct{}

// NewEncryptor returns the default encryptor.
// In the future, this can be extended to return different encryptors based on config.
func NewEncryptor() Encryptor {
	return &chaCha20Poly1305Encryptor{}
}

// deriveKey derives a key from the given password and salt using scrypt.
func (e *chaCha20Poly1305Encryptor) deriveKey(password string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keySize)
}

// Encrypt encrypts the given plaintext using ChaCha20-Poly1305 with a key derived from the password.
// The returned ciphertext includes the salt and nonce prepended.
func (e *chaCha20Poly1305Encryptor) Encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key, err := e.deriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD cipher: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// Prepend salt and nonce to the ciphertext
	result := append(salt, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// Decrypt decrypts the ciphertext using ChaCha20-Poly1305 with a key derived from the password.
// The input must have the salt and nonce prepended.
func (e *chaCha20Poly1305Encryptor) Decrypt(ciphertext []byte, password string) ([]byte, error) {
	if len(ciphertext) < saltSize+nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	salt := ciphertext[:saltSize]
	nonce := ciphertext[saltSize : saltSize+nonceSize]
	actualCiphertext := ciphertext[saltSize+nonceSize:]

	key, err := e.deriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD cipher: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// EncryptFile reads a file, encrypts its contents, and writing the source address of destination file.
func EncryptFile(e Encryptor, srcPath, dstPath, password string) error {
	in, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	encrypted, err := e.Encrypt(in, password)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	err = os.WriteFile(dstPath, encrypted, 0644)
	if err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

// DecryptFile reads a file, decrypts its contents, and writes it to the destination file.
func DecryptFile(e Encryptor, srcPath, dstPath, password string) error {
	in, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	decrypted, err := e.Decrypt(in, password)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	err = os.WriteFile(dstPath, decrypted, 0644)
	if err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	return nil
}
