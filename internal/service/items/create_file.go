package items

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/model"
	"golang.org/x/sync/errgroup"
)

var _ handlers.CreateFileService = (*CreateFileService)(nil)

const (
	fileChunkSize = 10 << 20
)

// FileEncryptor encrypts file streams and derived keys.
//
//go:generate mockgen -destination ./mocks/file_encryptor_gen.go -package mocks . FileEncryptor
type FileEncryptor interface {
	Encryptor
	NewFileNoncePrefix() ([]byte, error)
	EncryptFileStream(
		dek []byte,
		r io.Reader,
		w io.Writer,
		chunkSize int,
		noncePrefix []byte,
		aadPrefix []byte,
	) (int64, error)
}

// CreateFileService handles creation of encrypted file items and storage.
type CreateFileService struct {
	repo      model.ItemsRepository
	userRepo  model.UsersRepository
	encryptor FileEncryptor
	store     ObjectStorer
}

// NewCreateFileService constructs a CreateFileService instance.
func NewCreateFileService(
	repo model.ItemsRepository,
	userRepo model.UsersRepository,
	encryptor FileEncryptor,
	store ObjectStorer,
) *CreateFileService {
	return &CreateFileService{
		repo:      repo,
		userRepo:  userRepo,
		encryptor: encryptor,
		store:     store,
	}
}

// CreateFileItem encrypts file content and metadata, stores them, and creates the item entry.
func (s *CreateFileService) CreateFileItem(
	ctx context.Context,
	userID int,
	item *model.ItemDecrypted,
	fileContentReader multipart.File,
	contentType string,
) (int, error) {
	if item == nil || item.ItemMeta == nil {
		return 0, ErrInvalidItemData
	}
	if item.Type != model.ItemTypeFile {
		return 0, ErrInvalidItemType
	}

	var fd model.FileData
	if len(item.Data) > 0 {
		if err := json.Unmarshal(item.Data, &fd); err != nil {
			return 0, ErrInvalidFileData
		}
	}

	if contentType != "" {
		fd.ContentType = contentType
	}

	objectKey, err := newObjectKey()
	if err != nil {
		return 0, fmt.Errorf("generate object key: %w", err)
	}

	noncePrefix, err := s.encryptor.NewFileNoncePrefix()
	if err != nil {
		return 0, fmt.Errorf("generate file nonce prefix: %w", err)
	}

	edek, err := s.userRepo.GetUserDEK(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get user dek: %w", err)
	}

	dek, err := s.encryptor.DecryptDEK(edek)
	if err != nil {
		return 0, fmt.Errorf("decrypt user dek: %w", err)
	}

	pr, pw := io.Pipe()
	eg, egCtx := errgroup.WithContext(ctx)

	var plainSize int64

	eg.Go(func() error {
		defer func() { _ = pw.Close() }()
		n, err := s.encryptor.EncryptFileStream(
			dek,
			fileContentReader,
			pw,
			fileChunkSize,
			noncePrefix,
			[]byte(objectKey),
		)
		if err != nil {
			_ = pw.CloseWithError(err)
			return err
		}
		plainSize = n
		return nil
	})

	// Upload encrypted file stream to S3.
	// Note: s.store.Put does not retry because pr (io.Pipe reader) is not replayable.
	// If PutObject fails, the entire CreateFileItem operation will fail and should be
	// retried by the client if needed.
	eg.Go(func() error {
		return s.store.Put(egCtx, objectKey, pr)
	})

	if err := eg.Wait(); err != nil {
		return 0, fmt.Errorf("encrypt and store file: %w", err)
	}

	fd.Size = plainSize

	fdJSON, err := json.Marshal(&fd)
	if err != nil {
		return 0, fmt.Errorf("marshal file data: %w", err)
	}

	encMeta, nonce, err := s.encryptor.Encrypt(dek, fdJSON)
	if err != nil {
		return 0, fmt.Errorf("encrypt file data: %w", err)
	}

	itemEncrypted := &model.ItemEncrypted{
		ItemMeta:    item.ItemMeta,
		Ciphertext:  encMeta,
		ObjectKey:   objectKey,
		Nonce:       nonce,
		ChunkSize:   fileChunkSize,
		NoncePrefix: noncePrefix,
	}

	id, err := s.repo.CreateItem(ctx, userID, itemEncrypted)
	if err != nil {
		return 0, fmt.Errorf("create file item: %w", err)
	}

	return id, nil
}

func newObjectKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
