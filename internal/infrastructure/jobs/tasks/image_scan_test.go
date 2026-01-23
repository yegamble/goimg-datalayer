package tasks

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/clamav"
)

// --- Mocks ---

type MockScanner struct {
	mock.Mock
}

func (m *MockScanner) Scan(ctx context.Context, data []byte) (*clamav.ScanResult, error) {
	args := m.Called(ctx, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clamav.ScanResult), args.Error(1)
}

func (m *MockScanner) ScanReader(ctx context.Context, reader io.Reader, size int64) (*clamav.ScanResult, error) {
	args := m.Called(ctx, reader, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clamav.ScanResult), args.Error(1)
}

func (m *MockScanner) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockScanner) Version(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockScanner) Stats(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStorage) Put(ctx context.Context, key string, data []byte) error {
	args := m.Called(ctx, key, data)
	return args.Error(0)
}

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
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockImageRepository) FindPublic(ctx context.Context, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, pagination)
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockImageRepository) FindByTag(ctx context.Context, tag gallery.Tag, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, tag, pagination)
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockImageRepository) FindByStatus(ctx context.Context, status gallery.ImageStatus, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, status, pagination)
	return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

func (m *MockImageRepository) Search(ctx context.Context, params gallery.SearchParams) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, params)
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

func (m *MockImageRepository) ExistsByID(ctx context.Context, id gallery.ImageID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// --- Tests ---

func TestImageScanHandler_ProcessTask_Clean(t *testing.T) {
	// Arrange
	mockScanner := new(MockScanner)
	mockStorage := new(MockStorage)
	mockRepo := new(MockImageRepository)
	logger := zerolog.Nop()

	handler := NewImageScanHandler(mockScanner, mockStorage, mockRepo, logger)

	imageID := gallery.NewImageID()
	payload := ImageScanPayload{
		ImageID:          imageID.String(),
		StorageKey:       "images/test.jpg",
		OriginalFilename: "test.jpg",
		OwnerID:          identity.NewUserID().String(),
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeImageScan, payloadBytes)

	imageData := []byte("fake-image-data")

	// Create a valid image using ReconstructImage to control state
	// Need to setup metadata first
	metadata, _ := gallery.NewImageMetadata("Title", "Desc", "test.jpg", "image/jpeg", 100, 100, 1000, "key", "local")
	image := gallery.ReconstructImage(
		imageID,
		identity.NewUserID(),
		metadata,
		gallery.VisibilityPrivate,
		gallery.StatusProcessing,
		gallery.ScanStatusPending,
		[]gallery.ImageVariant{},
		[]gallery.Tag{},
		nil,
		0, 0, 0,
		time.Now(),
		time.Now(),
	)

	// Expectations
	mockScanner.On("Ping", mock.Anything).Return(nil)
	mockStorage.On("Get", mock.Anything, "images/test.jpg").Return(imageData, nil)
	mockRepo.On("FindByID", mock.Anything, imageID).Return(image, nil)

	scanResult := &clamav.ScanResult{
		Clean:     true,
		Infected:  false,
		ScannedAt: time.Now(),
	}
	mockScanner.On("Scan", mock.Anything, imageData).Return(scanResult, nil)

	mockRepo.On("Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
		return img.ScanStatus() == gallery.ScanStatusClean
	})).Return(nil)

	// Act
	err := handler.ProcessTask(context.Background(), task)

	// Assert
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockScanner.AssertExpectations(t)
}

func TestImageScanHandler_ProcessTask_Infected(t *testing.T) {
	// Arrange
	mockScanner := new(MockScanner)
	mockStorage := new(MockStorage)
	mockRepo := new(MockImageRepository)
	logger := zerolog.Nop()

	handler := NewImageScanHandler(mockScanner, mockStorage, mockRepo, logger)

	imageID := gallery.NewImageID()
	payload := ImageScanPayload{
		ImageID:          imageID.String(),
		StorageKey:       "images/malware.jpg",
		OriginalFilename: "malware.jpg",
		OwnerID:          identity.NewUserID().String(),
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeImageScan, payloadBytes)

	imageData := []byte("eicar-test-file")

	metadata, _ := gallery.NewImageMetadata("Title", "Desc", "malware.jpg", "image/jpeg", 100, 100, 1000, "key", "local")
	image := gallery.ReconstructImage(
		imageID,
		identity.NewUserID(),
		metadata,
		gallery.VisibilityPrivate,
		gallery.StatusProcessing,
		gallery.ScanStatusPending,
		[]gallery.ImageVariant{},
		[]gallery.Tag{},
		nil,
		0, 0, 0,
		time.Now(),
		time.Now(),
	)

	// Expectations
	mockScanner.On("Ping", mock.Anything).Return(nil)
	mockStorage.On("Get", mock.Anything, "images/malware.jpg").Return(imageData, nil)
	mockRepo.On("FindByID", mock.Anything, imageID).Return(image, nil)

	scanResult := &clamav.ScanResult{
		Clean:    false,
		Infected: true,
		Virus:    "Eicar-Test-Signature",
		ScannedAt: time.Now(),
	}
	mockScanner.On("Scan", mock.Anything, imageData).Return(scanResult, nil)

	mockRepo.On("Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
		return img.ScanStatus() == gallery.ScanStatusInfected
	})).Return(nil)

	// Act
	err := handler.ProcessTask(context.Background(), task)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "malware detected")

	mockRepo.AssertExpectations(t)
}
