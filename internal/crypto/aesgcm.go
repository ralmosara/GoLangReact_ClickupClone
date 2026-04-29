// Package crypto provides AES-256-GCM helpers for the credentials vault.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// KeySize is the required AES-256 key length in bytes.
const KeySize = 32

// ErrKeySize is returned when the provided key is not exactly KeySize bytes.
var ErrKeySize = errors.New("encryption key must be 32 bytes")

// Encrypt seals plaintext with AES-256-GCM using a fresh random 12-byte nonce.
// Returns the ciphertext (with the GCM auth tag appended) and the nonce
// separately so the caller can store them in distinct columns.
func Encrypt(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	if len(key) != KeySize {
		return nil, nil, ErrKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt opens an AES-256-GCM ciphertext+nonce pair produced by Encrypt.
// Returns an error if the key is wrong, the nonce is wrong, or the ciphertext
// has been tampered with.
func Decrypt(key, ciphertext, nonce []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid nonce length")
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}
