package cryptor

import (
	"bytes"
	"testing"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCryptor(t *testing.T) {
	t.Run("creates cryptor with master key", func(t *testing.T) {
		c := NewCryptor("test_master_key")
		assert.NotNil(t, c)
	})

	t.Run("creates cryptor with empty key", func(t *testing.T) {
		c := NewCryptor("")
		assert.NotNil(t, c)
	})

	t.Run("creates cryptor with long key", func(t *testing.T) {
		c := NewCryptor("very_long_master_key_for_testing_purposes_12345")
		assert.NotNil(t, c)
	})
}

func TestCryptor_CreateDEK(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		dek, err := c.CreateDEK()

		require.NoError(t, err)
		assert.NotNil(t, dek)
		assert.NotEmpty(t, dek.EncryptedKey)
		assert.NotEmpty(t, dek.Nonce)
		assert.Len(t, dek.Nonce, 12) // GCM nonce size
	})

	t.Run("different keys each time", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		dek1, err := c.CreateDEK()
		require.NoError(t, err)

		dek2, err := c.CreateDEK()
		require.NoError(t, err)

		assert.NotEqual(t, dek1.EncryptedKey, dek2.EncryptedKey)
		assert.NotEqual(t, dek1.Nonce, dek2.Nonce)
	})
}

func TestCryptor_DecryptDEK(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		dek, err := c.DecryptDEK(edek)

		require.NoError(t, err)
		assert.NotEmpty(t, dek)
		assert.Len(t, dek, 32) // 32 bytes DEK
	})

	t.Run("wrong master key", func(t *testing.T) {
		c1 := NewCryptor("master_key_1")
		c2 := NewCryptor("master_key_2")

		edek, err := c1.CreateDEK()
		require.NoError(t, err)

		_, err = c2.DecryptDEK(edek)

		assert.Error(t, err)
	})

	t.Run("invalid encrypted data", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		invalidDEK := &model.EncryptedDEK{
			EncryptedKey: []byte("invalid_encrypted_data"),
			Nonce:        edek.Nonce, // valid nonce length
		}

		_, err = c.DecryptDEK(invalidDEK)

		assert.Error(t, err)
	})
}

func TestCryptor_Encrypt_Decrypt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		dek, err := c.DecryptDEK(edek)
		require.NoError(t, err)

		plaintext := []byte("sensitive data to encrypt")

		ciphertext, nonce, err := c.Encrypt(dek, plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
		assert.NotEmpty(t, nonce)
		assert.NotEqual(t, plaintext, ciphertext)

		decrypted, err := c.Decrypt(dek, nonce, ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("empty data", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		dek, err := c.DecryptDEK(edek)
		require.NoError(t, err)

		plaintext := []byte{}

		ciphertext, nonce, err := c.Encrypt(dek, plaintext)
		require.NoError(t, err)

		decrypted, err := c.Decrypt(dek, nonce, ciphertext)
		require.NoError(t, err)
		assert.Empty(t, decrypted)
	})

	t.Run("wrong nonce", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		dek, err := c.DecryptDEK(edek)
		require.NoError(t, err)

		plaintext := []byte("sensitive data")

		ciphertext, _, err := c.Encrypt(dek, plaintext)
		require.NoError(t, err)

		wrongNonce := make([]byte, 12)

		_, err = c.Decrypt(dek, wrongNonce, ciphertext)
		assert.Error(t, err)
	})

	t.Run("wrong key", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek1, err := c.CreateDEK()
		require.NoError(t, err)

		dek1, err := c.DecryptDEK(edek1)
		require.NoError(t, err)

		edek2, err := c.CreateDEK()
		require.NoError(t, err)

		dek2, err := c.DecryptDEK(edek2)
		require.NoError(t, err)

		plaintext := []byte("sensitive data")

		ciphertext, nonce, err := c.Encrypt(dek1, plaintext)
		require.NoError(t, err)

		_, err = c.Decrypt(dek2, nonce, ciphertext)
		assert.Error(t, err)
	})

	t.Run("invalid key length for encrypt", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		invalidDEK := []byte("short")
		plaintext := []byte("test data")

		_, _, err := c.Encrypt(invalidDEK, plaintext)
		assert.Error(t, err)
	})

	t.Run("invalid key length for decrypt", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		invalidDEK := []byte("short")
		nonce := make([]byte, 12)
		ciphertext := []byte("encrypted")

		_, err := c.Decrypt(invalidDEK, nonce, ciphertext)
		assert.Error(t, err)
	})
}

func TestCryptor_EncryptFileStream(t *testing.T) {
	t.Run("success small file", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		dek, err := c.DecryptDEK(edek)
		require.NoError(t, err)

		plaintext := []byte("Hello, World! This is a test file.")
		noncePrefix, err := generateRandom(8)
		require.NoError(t, err)

		aadPrefix := []byte("file_metadata")
		chunkSize := 16

		reader := bytes.NewReader(plaintext)
		var encrypted bytes.Buffer

		totalBytes, err := c.EncryptFileStream(
			dek,
			reader,
			&encrypted,
			chunkSize,
			noncePrefix,
			aadPrefix,
		)

		require.NoError(t, err)
		assert.Equal(t, int64(len(plaintext)), totalBytes)
		assert.Greater(
			t,
			encrypted.Len(),
			len(plaintext),
		) // encrypted data is larger due to auth tag
	})

	t.Run("invalid nonce prefix size", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		dek, err := c.DecryptDEK(edek)
		require.NoError(t, err)

		plaintext := []byte("test data")
		wrongNoncePrefix := []byte("short") // должно быть 8 байт

		reader := bytes.NewReader(plaintext)
		var encrypted bytes.Buffer

		_, err = c.EncryptFileStream(
			dek,
			reader,
			&encrypted,
			16,
			wrongNoncePrefix,
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid nonce prefix size")
	})

	t.Run("invalid chunk size", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		dek, err := c.DecryptDEK(edek)
		require.NoError(t, err)

		plaintext := []byte("test data")
		noncePrefix, err := generateRandom(8)
		require.NoError(t, err)

		reader := bytes.NewReader(plaintext)
		var encrypted bytes.Buffer

		_, err = c.EncryptFileStream(dek, reader, &encrypted, 0, noncePrefix, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid chunk size")
	})
}

func TestCryptor_DecryptFileStream(t *testing.T) {
	t.Run("success round trip", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		dek, err := c.DecryptDEK(edek)
		require.NoError(t, err)

		originalData := []byte("Hello, World! This is a test file for encryption.")
		noncePrefix, err := generateRandom(8)
		require.NoError(t, err)

		aadPrefix := []byte("file_metadata")
		chunkSize := 16

		// Encrypt
		reader := bytes.NewReader(originalData)
		var encrypted bytes.Buffer

		totalBytes, err := c.EncryptFileStream(
			dek,
			reader,
			&encrypted,
			chunkSize,
			noncePrefix,
			aadPrefix,
		)
		require.NoError(t, err)
		assert.Equal(t, int64(len(originalData)), totalBytes)

		// Decrypt
		var decrypted bytes.Buffer
		err = c.DecryptFileStream(
			dek,
			&encrypted,
			&decrypted,
			chunkSize,
			noncePrefix,
			aadPrefix,
		)

		require.NoError(t, err)
		assert.Equal(t, originalData, decrypted.Bytes())
	})

	t.Run("wrong key", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek1, err := c.CreateDEK()
		require.NoError(t, err)

		dek1, err := c.DecryptDEK(edek1)
		require.NoError(t, err)

		edek2, err := c.CreateDEK()
		require.NoError(t, err)

		dek2, err := c.DecryptDEK(edek2)
		require.NoError(t, err)

		originalData := []byte("secret file content")
		noncePrefix, err := generateRandom(8)
		require.NoError(t, err)

		chunkSize := 16

		// Encrypt with dek1
		reader := bytes.NewReader(originalData)
		var encrypted bytes.Buffer

		_, err = c.EncryptFileStream(
			dek1,
			reader,
			&encrypted,
			chunkSize,
			noncePrefix,
			nil,
		)
		require.NoError(t, err)

		// Try to decrypt with dek2
		var decrypted bytes.Buffer
		err = c.DecryptFileStream(
			dek2,
			&encrypted,
			&decrypted,
			chunkSize,
			noncePrefix,
			nil,
		)

		assert.Error(t, err)
	})

	t.Run("empty file", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		edek, err := c.CreateDEK()
		require.NoError(t, err)

		dek, err := c.DecryptDEK(edek)
		require.NoError(t, err)

		originalData := []byte{}
		noncePrefix, err := generateRandom(8)
		require.NoError(t, err)

		chunkSize := 16

		// Encrypt
		reader := bytes.NewReader(originalData)
		var encrypted bytes.Buffer

		totalBytes, err := c.EncryptFileStream(
			dek,
			reader,
			&encrypted,
			chunkSize,
			noncePrefix,
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, int64(0), totalBytes)

		// Decrypt
		var decrypted bytes.Buffer
		err = c.DecryptFileStream(
			dek,
			&encrypted,
			&decrypted,
			chunkSize,
			noncePrefix,
			nil,
		)

		require.NoError(t, err)
		assert.Empty(t, decrypted.Bytes())
	})
}

func TestGenerateRandom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		size := 32
		data, err := generateRandom(size)

		require.NoError(t, err)
		assert.Len(t, data, size)
	})

	t.Run("different each time", func(t *testing.T) {
		data1, err := generateRandom(16)
		require.NoError(t, err)

		data2, err := generateRandom(16)
		require.NoError(t, err)

		assert.NotEqual(t, data1, data2)
	})

	t.Run("zero size", func(t *testing.T) {
		data, err := generateRandom(0)

		require.NoError(t, err)
		assert.Empty(t, data)
	})
}

func TestCryptor_NewFileNoncePrefix(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		prefix, err := c.NewFileNoncePrefix()

		require.NoError(t, err)
		assert.Len(t, prefix, 8)
	})

	t.Run("different each time", func(t *testing.T) {
		c := NewCryptor("test_master_key")

		prefix1, err := c.NewFileNoncePrefix()
		require.NoError(t, err)

		prefix2, err := c.NewFileNoncePrefix()
		require.NoError(t, err)

		assert.NotEqual(t, prefix1, prefix2)
	})
}
