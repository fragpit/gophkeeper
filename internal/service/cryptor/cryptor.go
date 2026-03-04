package cryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/fragpit/gophkeeper/internal/service/items"
)

var _ auth.DEKCreator = (*Cryptor)(nil)
var _ items.Encryptor = (*Cryptor)(nil)

var _ items.FileEncryptor = (*Cryptor)(nil)
var _ items.FileDecryptor = (*Cryptor)(nil)

// Cryptor implements encryption and decryption helpers using AES-GCM.
type Cryptor struct {
	masterKey string
}

// NewCryptor creates a Cryptor instance bound to the provided master key.
func NewCryptor(masterKey string) *Cryptor {
	return &Cryptor{
		masterKey: masterKey,
	}
}

// CreateDEK generates a new encrypted data encryption key using the master key.
func (c *Cryptor) CreateDEK() (*model.EncryptedDEK, error) {
	key := sha256.Sum256([]byte(c.masterKey))

	aesBlock, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create new cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("create new gcm cipher: %w", err)
	}

	nonce, err := generateRandom(aesgcm.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("generate random nonce: %w", err)
	}

	randKey, err := generateRandom(32)
	if err != nil {
		return nil, fmt.Errorf("generate random user dek: %w", err)
	}

	encryptedDEK := aesgcm.Seal(nil, nonce, randKey, nil)

	return &model.EncryptedDEK{
		EncryptedKey: encryptedDEK,
		Nonce:        nonce,
	}, nil
}

// DecryptDEK decrypts an encrypted DEK using the master key.
func (c *Cryptor) DecryptDEK(edek *model.EncryptedDEK) ([]byte, error) {
	key := sha256.Sum256([]byte(c.masterKey))

	aesBlock, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create new cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("create new gcm cipher: %w", err)
	}

	plainDEK, err := aesgcm.Open(nil, edek.Nonce, edek.EncryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt dek: %w", err)
	}

	return plainDEK, nil
}

// Encrypt encrypts arbitrary data with the provided DEK.
func (c *Cryptor) Encrypt(dek, data []byte) ([]byte, []byte, error) {
	aesBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, nil, fmt.Errorf("create new cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, nil, fmt.Errorf("create new gcm cipher: %w", err)
	}

	nonce, err := generateRandom(aesgcm.NonceSize())
	if err != nil {
		return nil, nil, fmt.Errorf("generate random nonce: %w", err)
	}

	encrypted := aesgcm.Seal(nil, nonce, data, nil)
	return encrypted, nonce, nil
}

// Decrypt decrypts ciphertext with the provided DEK and nonce.
func (c *Cryptor) Decrypt(dek, nonce, data []byte) ([]byte, error) {
	aesBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("create new cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("create new gcm cipher: %w", err)
	}

	decrypted, err := aesgcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt data: %w", err)
	}

	return decrypted, nil
}

// EncryptFileStream encrypts a streaming reader into writer using chunked AES-GCM.
func (c *Cryptor) EncryptFileStream(
	dek []byte,
	r io.Reader,
	w io.Writer,
	chunkSize int,
	noncePrefix []byte,
	aadPrefix []byte,
) (int64, error) {
	aesBlock, err := aes.NewCipher(dek)
	if err != nil {
		return 0, fmt.Errorf("create new cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return 0, fmt.Errorf("create new gcm cipher: %w", err)
	}

	nonceSuffixSize := 4
	if len(noncePrefix) != aesgcm.NonceSize()-nonceSuffixSize {
		return 0, fmt.Errorf("invalid nonce prefix size")
	}

	if chunkSize <= 0 {
		return 0, fmt.Errorf("invalid chunk size")
	}

	buf := make([]byte, chunkSize)
	var total int64
	var counter uint32

	for {
		n, readErr := io.ReadFull(r, buf)
		if readErr == io.EOF {
			break
		}
		if readErr == io.ErrUnexpectedEOF && n == 0 {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return total, fmt.Errorf("read chunk: %w", readErr)
		}

		nonce := make([]byte, aesgcm.NonceSize())
		copy(nonce, noncePrefix)
		binary.BigEndian.PutUint32(nonce[len(nonce)-nonceSuffixSize:], counter)

		aad := make([]byte, len(aadPrefix)+nonceSuffixSize)
		copy(aad, aadPrefix)
		binary.BigEndian.PutUint32(aad[len(aad)-nonceSuffixSize:], counter)

		encrypted := aesgcm.Seal(nil, nonce, buf[:n], aad)
		if _, err := w.Write(encrypted); err != nil {
			return total, fmt.Errorf("write encrypted chunk: %w", err)
		}

		total += int64(n)
		counter++

		if readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	return total, nil
}

// DecryptFileStream decrypts an encrypted stream written by EncryptFileStream.
func (c *Cryptor) DecryptFileStream(
	dek []byte,
	r io.Reader,
	w io.Writer,
	chunkSize int,
	noncePrefix []byte,
	aadPrefix []byte,
) error {
	aesBlock, err := aes.NewCipher(dek)
	if err != nil {
		return fmt.Errorf("create new cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return fmt.Errorf("create new gcm cipher: %w", err)
	}

	nonceSuffixSize := 4
	if len(noncePrefix) != aesgcm.NonceSize()-nonceSuffixSize {
		return fmt.Errorf("invalid nonce prefix size")
	}

	if chunkSize <= 0 {
		return fmt.Errorf("invalid chunk size")
	}

	cipherChunkSize := chunkSize + aesgcm.Overhead()
	buf := make([]byte, cipherChunkSize)
	var counter uint32

	for {
		n, readErr := io.ReadFull(r, buf)
		if readErr == io.EOF {
			break
		}
		if readErr == io.ErrUnexpectedEOF && n == 0 {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return fmt.Errorf("read chunk: %w", readErr)
		}
		if n < aesgcm.Overhead() {
			return fmt.Errorf("read chunk: ciphertext too small")
		}

		nonce := make([]byte, aesgcm.NonceSize())
		copy(nonce, noncePrefix)
		binary.BigEndian.PutUint32(nonce[len(nonce)-nonceSuffixSize:], counter)

		aad := make([]byte, len(aadPrefix)+nonceSuffixSize)
		copy(aad, aadPrefix)
		binary.BigEndian.PutUint32(aad[len(aad)-nonceSuffixSize:], counter)

		plain, err := aesgcm.Open(nil, nonce, buf[:n], aad)
		if err != nil {
			return fmt.Errorf("decrypt data: %w", err)
		}

		if _, err := w.Write(plain); err != nil {
			return fmt.Errorf("write decrypted chunk: %w", err)
		}

		counter++

		if readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	return nil
}

// NewFileNoncePrefix generates a nonce prefix for file encryption operations.
func (c *Cryptor) NewFileNoncePrefix() ([]byte, error) {
	return generateRandom(8)
}

func generateRandom(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
