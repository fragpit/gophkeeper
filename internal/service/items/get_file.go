package items

import (
	"context"
	"fmt"
	"io"

	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/model"
)

var _ handlers.GetFileService = (*GetFileService)(nil)

// FileDecryptor decrypts file streams using provided keys.
//
//go:generate mockgen -destination ./mocks/file_decryptor_gen.go -package mocks . FileDecryptor
type FileDecryptor interface {
	Encryptor
	DecryptFileStream(
		dek []byte,
		r io.Reader,
		w io.Writer,
		chunkSize int,
		noncePrefix []byte,
		aadPrefix []byte,
	) error
}

// GetFileService retrieves encrypted items and decrypts their file payloads.
type GetFileService struct {
	repo     model.ItemsRepository
	userRepo model.UsersRepository
	crypt    FileDecryptor
	store    ObjectStorer
}

// NewGetFileService constructs a GetFileService with the given dependencies.
func NewGetFileService(
	repo model.ItemsRepository,
	userRepo model.UsersRepository,
	crypt FileDecryptor,
	store ObjectStorer,
) *GetFileService {
	return &GetFileService{
		repo:     repo,
		userRepo: userRepo,
		crypt:    crypt,
		store:    store,
	}
}

// GetItemByTitle fetches an encrypted item by its title for the specified user.
func (s *GetFileService) GetItemByTitle(
	ctx context.Context,
	userID int,
	title string,
) (*model.ItemEncrypted, error) {
	return s.repo.GetItemByTitle(ctx, userID, title)
}

// GetFileByItem retrieves and decrypts file content for the provided item.
func (s *GetFileService) GetFileByItem(
	ctx context.Context,
	userID int,
	item *model.ItemEncrypted,
) (io.ReadCloser, error) {
	if item == nil || item.ItemMeta == nil {
		return nil, ErrInvalidItemData
	}
	if item.Type != model.ItemTypeFile {
		return nil, ErrInvalidItemType
	}

	if item.ObjectKey == "" {
		return nil, ErrEmptyObjectKey
	}
	if len(item.NoncePrefix) == 0 {
		return nil, ErrEmptyNoncePrefix
	}
	if item.ChunkSize <= 0 {
		return nil, ErrEmptyChunkSize
	}

	edek, err := s.userRepo.GetUserDEK(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user dek: %w", err)
	}

	dek, err := s.crypt.DecryptDEK(edek)
	if err != nil {
		return nil, fmt.Errorf("decrypt user dek: %w", err)
	}

	ctx2, cancel := context.WithCancel(ctx)
	obj, err := s.store.Get(ctx2, "gophkeeper", item.ObjectKey)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("get s3 file: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		defer func() { _ = obj.Close() }()

		err := s.crypt.DecryptFileStream(
			dek,
			obj,
			pw,
			item.ChunkSize,
			item.NoncePrefix,
			[]byte(item.ObjectKey),
		)

		_ = pw.CloseWithError(err)
	}()

	return &fileStream{r: pr, cancel: cancel, obj: obj}, nil
}

type fileStream struct {
	r      *io.PipeReader
	cancel context.CancelFunc
	obj    io.Closer
}

func (s *fileStream) Read(p []byte) (int, error) { return s.r.Read(p) }

func (s *fileStream) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.obj != nil {
		_ = s.obj.Close()
	}
	return s.r.Close()
}
