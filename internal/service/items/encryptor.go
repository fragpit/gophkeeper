package items

import "github.com/fragpit/gophkeeper/internal/model"

type Encryptor interface {
	Encrypt(dek, data []byte) (encryptedData []byte, nonce []byte, err error)
	Decrypt(dek, nonce, data []byte) (decryptedData []byte, err error)
	DecryptDEK(edek *model.EncryptedDEK) (dek []byte, err error)
}
