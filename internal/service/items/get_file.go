package items

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/model"
)

var _ handlers.GetFileService = (*GetFileService)(nil)

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

type GetFileService struct {
	repo     model.ItemsRepository
	userRepo model.UsersRepository
	crypt    FileDecryptor
	store    ObjectStorer
}

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

func (s *GetFileService) GetItemByTitle(
	ctx context.Context,
	userID int,
	title string,
) (*model.ItemEncrypted, error) {
	return s.repo.GetItemByTitle(ctx, userID, title)
}

func (s *GetFileService) GetFileByItem(
	ctx context.Context,
	userID int,
	item *model.ItemEncrypted,
) (io.ReadCloser, error) {
	if item == nil || item.ItemMeta == nil {
		return nil, errors.New("invalid item data")
	}
	if item.Type != model.ItemTypeFile {
		return nil, errors.New("invalid item type")
	}

	if item.ObjectKey == "" {
		return nil, errors.New("empty object key")
	}
	if len(item.NoncePrefix) == 0 {
		return nil, errors.New("empty nonce prefix")
	}
	if item.ChunkSize <= 0 {
		return nil, errors.New("empty chunk size")
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
		defer obj.Close()

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
