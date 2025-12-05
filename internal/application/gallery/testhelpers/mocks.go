package testhelpers

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
)

// MockImageRepository is a mock implementation of gallery.ImageRepository.
type MockImageRepository struct {
	mock.Mock
}

func (m *MockImageRepository) NextID() gallery.ImageID {
	args := m.Called()
	return args.Get(0).(gallery.ImageID)
}

func (m *MockImageRepository) FindByID(ctx context.Context, id gallery.ImageID) (*gallery.Image, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gallery.Image), args.Error(1)
}

func (m *MockImageRepository) FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, ownerID, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockImageRepository) FindPublic(ctx context.Context, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockImageRepository) FindByTag(ctx context.Context, tag gallery.Tag, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, tag, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockImageRepository) FindByStatus(ctx context.Context, status gallery.ImageStatus, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, status, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockImageRepository) Save(ctx context.Context, image *gallery.Image) error {
	args := m.Called(ctx, image)
	return args.Error(0)
}

func (m *MockImageRepository) Delete(ctx context.Context, id gallery.ImageID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockImageRepository) Search(ctx context.Context, params gallery.SearchParams) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockImageRepository) ExistsByID(ctx context.Context, id gallery.ImageID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// MockStorage is a mock implementation of storage.Storage.
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Put(ctx context.Context, key string, data io.Reader, size int64, opts storage.PutOptions) error {
	args := m.Called(ctx, key, data, size, opts)
	return args.Error(0)
}

func (m *MockStorage) PutBytes(ctx context.Context, key string, data []byte, opts storage.PutOptions) error {
	args := m.Called(ctx, key, data, opts)
	return args.Error(0)
}

func (m *MockStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorage) GetBytes(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorage) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorage) URL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockStorage) PresignedURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	args := m.Called(ctx, key, duration)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Stat(ctx context.Context, key string) (*storage.ObjectInfo, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ObjectInfo), args.Error(1)
}

func (m *MockStorage) Provider() string {
	args := m.Called()
	return args.String(0)
}

// MockJobEnqueuer is a mock implementation of JobEnqueuer.
type MockJobEnqueuer struct {
	mock.Mock
}

func (m *MockJobEnqueuer) EnqueueImageProcessing(ctx context.Context, imageID string) error {
	args := m.Called(ctx, imageID)
	return args.Error(0)
}

func (m *MockJobEnqueuer) EnqueueImageCleanup(ctx context.Context, imageID, storageProvider string, keys []string) error {
	args := m.Called(ctx, imageID, storageProvider, keys)
	return args.Error(0)
}

// MockEventPublisher is a mock implementation of EventPublisher.
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockAlbumRepository is a mock implementation of gallery.AlbumRepository.
type MockAlbumRepository struct {
	mock.Mock
}

func (m *MockAlbumRepository) NextID() gallery.AlbumID {
	args := m.Called()
	return args.Get(0).(gallery.AlbumID)
}

func (m *MockAlbumRepository) FindByID(ctx context.Context, id gallery.AlbumID) (*gallery.Album, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gallery.Album), args.Error(1)
}

func (m *MockAlbumRepository) FindByOwner(ctx context.Context, ownerID identity.UserID) ([]*gallery.Album, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gallery.Album), args.Error(1)
}

func (m *MockAlbumRepository) FindPublic(ctx context.Context, pagination shared.Pagination) ([]*gallery.Album, int64, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Album), args.Get(1).(int64), args.Error(2)
}

func (m *MockAlbumRepository) Save(ctx context.Context, album *gallery.Album) error {
	args := m.Called(ctx, album)
	return args.Error(0)
}

func (m *MockAlbumRepository) Delete(ctx context.Context, id gallery.AlbumID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAlbumRepository) ExistsByID(ctx context.Context, id gallery.AlbumID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// MockCommentRepository is a mock implementation of gallery.CommentRepository.
type MockCommentRepository struct {
	mock.Mock
}

func (m *MockCommentRepository) NextID() gallery.CommentID {
	args := m.Called()
	return args.Get(0).(gallery.CommentID)
}

func (m *MockCommentRepository) FindByID(ctx context.Context, id gallery.CommentID) (*gallery.Comment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gallery.Comment), args.Error(1)
}

func (m *MockCommentRepository) FindByImage(ctx context.Context, imageID gallery.ImageID, pagination shared.Pagination) ([]*gallery.Comment, int64, error) {
	args := m.Called(ctx, imageID, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Comment), args.Get(1).(int64), args.Error(2)
}

func (m *MockCommentRepository) FindByUser(ctx context.Context, userID identity.UserID, pagination shared.Pagination) ([]*gallery.Comment, int64, error) {
	args := m.Called(ctx, userID, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Comment), args.Get(1).(int64), args.Error(2)
}

func (m *MockCommentRepository) CountByImage(ctx context.Context, imageID gallery.ImageID) (int64, error) {
	args := m.Called(ctx, imageID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCommentRepository) Save(ctx context.Context, comment *gallery.Comment) error {
	args := m.Called(ctx, comment)
	return args.Error(0)
}

func (m *MockCommentRepository) Delete(ctx context.Context, id gallery.CommentID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCommentRepository) ExistsByID(ctx context.Context, id gallery.CommentID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// MockLikeRepository is a mock implementation of gallery.LikeRepository.
type MockLikeRepository struct {
	mock.Mock
}

func (m *MockLikeRepository) Like(ctx context.Context, userID identity.UserID, imageID gallery.ImageID) error {
	args := m.Called(ctx, userID, imageID)
	return args.Error(0)
}

func (m *MockLikeRepository) Unlike(ctx context.Context, userID identity.UserID, imageID gallery.ImageID) error {
	args := m.Called(ctx, userID, imageID)
	return args.Error(0)
}

func (m *MockLikeRepository) HasLiked(ctx context.Context, userID identity.UserID, imageID gallery.ImageID) (bool, error) {
	args := m.Called(ctx, userID, imageID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLikeRepository) GetLikeCount(ctx context.Context, imageID gallery.ImageID) (int64, error) {
	args := m.Called(ctx, imageID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockLikeRepository) GetLikedImageIDs(ctx context.Context, userID identity.UserID, pagination shared.Pagination) ([]uuid.UUID, error) {
	args := m.Called(ctx, userID, pagination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockLikeRepository) CountLikedImagesByUser(ctx context.Context, userID identity.UserID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// MockAlbumImageRepository is a mock implementation of gallery.AlbumImageRepository.
type MockAlbumImageRepository struct {
	mock.Mock
}

func (m *MockAlbumImageRepository) AddImageToAlbum(ctx context.Context, albumID gallery.AlbumID, imageID gallery.ImageID) error {
	args := m.Called(ctx, albumID, imageID)
	return args.Error(0)
}

func (m *MockAlbumImageRepository) RemoveImageFromAlbum(ctx context.Context, albumID gallery.AlbumID, imageID gallery.ImageID) error {
	args := m.Called(ctx, albumID, imageID)
	return args.Error(0)
}

func (m *MockAlbumImageRepository) FindImagesInAlbum(ctx context.Context, albumID gallery.AlbumID, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, albumID, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockAlbumImageRepository) FindAlbumsForImage(ctx context.Context, imageID gallery.ImageID) ([]*gallery.Album, error) {
	args := m.Called(ctx, imageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gallery.Album), args.Error(1)
}

func (m *MockAlbumImageRepository) IsImageInAlbum(ctx context.Context, albumID gallery.AlbumID, imageID gallery.ImageID) (bool, error) {
	args := m.Called(ctx, albumID, imageID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAlbumImageRepository) CountImagesInAlbum(ctx context.Context, albumID gallery.AlbumID) (int, error) {
	args := m.Called(ctx, albumID)
	return args.Int(0), args.Error(1)
}

// MockUserRepository is a mock implementation of identity.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) NextID() identity.UserID {
	args := m.Called()
	return args.Get(0).(identity.UserID)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id identity.UserID) (*identity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username identity.Username) (*identity.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id identity.UserID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) ExistsByID(ctx context.Context, id identity.UserID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}
