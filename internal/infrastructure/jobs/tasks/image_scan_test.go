package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/clamav"
)

// MockScanner is a mock implementation of clamav.Scanner.
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

// MockTaskStorage is a mock implementation of tasks.Storage.
type MockTaskStorage struct {
	mock.Mock
}

func (m *MockTaskStorage) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTaskStorage) Put(ctx context.Context, key string, data []byte) error {
	args := m.Called(ctx, key, data)
	return args.Error(0)
}

func TestImageScanHandler_ProcessTask(t *testing.T) {
	logger := zerolog.Nop()

	// Helper to create task
	createTask := func(payload ImageScanPayload) *asynq.Task {
		data, _ := json.Marshal(payload)
		return asynq.NewTask(TypeImageScan, data)
	}

	validImageID := gallery.NewImageID()
	ownerID := identity.NewUserID()
	validPayload := ImageScanPayload{
		ImageID:          validImageID.String(),
		StorageKey:       "images/123/original",
		OriginalFilename: "test.jpg",
		OwnerID:          ownerID.String(),
	}

	t.Run("successful clean scan", func(t *testing.T) {
		// Arrange
		mockRepo := new(testhelpers.MockImageRepository)
		mockScanner := new(MockScanner)
		mockStorage := new(MockTaskStorage)

		handler := NewImageScanHandler(mockRepo, mockScanner, mockStorage, logger)
		task := createTask(validPayload)

		// Expectations
		mockScanner.On("Ping", mock.Anything).Return(nil)
		mockStorage.On("Get", mock.Anything, validPayload.StorageKey).
			Return([]byte("fake-image-data"), nil)
		mockScanner.On("Scan", mock.Anything, []byte("fake-image-data")).
			Return(&clamav.ScanResult{Clean: true, ScannedAt: time.Now()}, nil)

		// Repo expectations
		mockImage, err := gallery.NewImageWithID(validImageID, ownerID, gallery.ImageMetadata{})
		require.NoError(t, err)
		mockRepo.On("FindByID", mock.Anything, validImageID).Return(mockImage, nil)
		mockRepo.On("Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
			return img.ID() == validImageID && img.Status() == gallery.StatusActive
		})).Return(nil)

		// Act
		err = handler.ProcessTask(context.Background(), task)

		// Assert
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockScanner.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})

	t.Run("infected scan", func(t *testing.T) {
		// Arrange
		mockRepo := new(testhelpers.MockImageRepository)
		mockScanner := new(MockScanner)
		mockStorage := new(MockTaskStorage)

		handler := NewImageScanHandler(mockRepo, mockScanner, mockStorage, logger)
		task := createTask(validPayload)

		// Expectations
		mockScanner.On("Ping", mock.Anything).Return(nil)
		mockStorage.On("Get", mock.Anything, validPayload.StorageKey).
			Return([]byte("infected-data"), nil)
		mockScanner.On("Scan", mock.Anything, []byte("infected-data")).
			Return(&clamav.ScanResult{Infected: true, Virus: "EICAR", ScannedAt: time.Now()}, nil)

		// Repo should NOT be called for finding/saving active image (TODOs handle this case)
		// Act
		err := handler.ProcessTask(context.Background(), task)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "malware detected")
		mockRepo.AssertNotCalled(t, "FindByID")
		mockRepo.AssertNotCalled(t, "Save")
	})

	t.Run("repository find error", func(t *testing.T) {
		// Arrange
		mockRepo := new(testhelpers.MockImageRepository)
		mockScanner := new(MockScanner)
		mockStorage := new(MockTaskStorage)

		handler := NewImageScanHandler(mockRepo, mockScanner, mockStorage, logger)
		task := createTask(validPayload)

		// Expectations
		mockScanner.On("Ping", mock.Anything).Return(nil)
		mockStorage.On("Get", mock.Anything, validPayload.StorageKey).
			Return([]byte("clean-data"), nil)
		mockScanner.On("Scan", mock.Anything, []byte("clean-data")).
			Return(&clamav.ScanResult{Clean: true, ScannedAt: time.Now()}, nil)

		// Repo returns error
		mockRepo.On("FindByID", mock.Anything, validImageID).
			Return(nil, errors.New("db error"))

		// Act
		err := handler.ProcessTask(context.Background(), task)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "find image")
	})

	t.Run("repository save error", func(t *testing.T) {
		// Arrange
		mockRepo := new(testhelpers.MockImageRepository)
		mockScanner := new(MockScanner)
		mockStorage := new(MockTaskStorage)

		handler := NewImageScanHandler(mockRepo, mockScanner, mockStorage, logger)
		task := createTask(validPayload)

		// Expectations
		mockScanner.On("Ping", mock.Anything).Return(nil)
		mockStorage.On("Get", mock.Anything, validPayload.StorageKey).
			Return([]byte("clean-data"), nil)
		mockScanner.On("Scan", mock.Anything, []byte("clean-data")).
			Return(&clamav.ScanResult{Clean: true, ScannedAt: time.Now()}, nil)

		mockImage, err := gallery.NewImageWithID(validImageID, ownerID, gallery.ImageMetadata{})
		require.NoError(t, err)
		mockRepo.On("FindByID", mock.Anything, validImageID).Return(mockImage, nil)
		mockRepo.On("Save", mock.Anything, mock.Anything).
			Return(errors.New("save error"))

		// Act
		err = handler.ProcessTask(context.Background(), task)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save image")
	})
}
