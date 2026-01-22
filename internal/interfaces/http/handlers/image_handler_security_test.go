package handlers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// MockStorage is a mock implementation of storage.Storage
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
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorage) GetBytes(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
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

// MockJobEnqueuer is a mock implementation of appgallery.JobEnqueuer
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

// MockEventPublisher is a mock implementation of appgallery.EventPublisher
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// Helper to create a multipart request
func createMultipartRequest(t *testing.T, fieldName, filename, contentType string, content []byte) (*http.Request, string) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Create a part with custom Content-Type
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="` + fieldName + `"; filename="` + filename + `"`}
	h["Content-Type"] = []string{contentType}
	part, err := writer.CreatePart(h)
	assert.NoError(t, err)
	_, err = part.Write(content)
	assert.NoError(t, err)

	// Add other required fields
	assert.NoError(t, writer.WriteField("title", "Test Image"))

	assert.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, writer.FormDataContentType()
}

func TestImageHandler_Upload_MimeTypeSpoofing(t *testing.T) {
	// Arrange
	mockRepo := new(MockImageRepository)
	mockStorage := new(MockStorage)
	mockJobEnqueuer := new(MockJobEnqueuer)
	mockEventPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()
	loggerPtr := &logger

	// Create command handler with mocks
	uploadHandler := commands.NewUploadImageHandler(
		mockRepo,
		mockStorage,
		mockJobEnqueuer,
		mockEventPublisher,
		loggerPtr,
	)

	// Create image handler
	handler := NewImageHandler(
		uploadHandler,
		nil, nil, nil, nil, nil, nil,
		mockStorage,
		logger,
	)

	// Expectations: NO storage calls, NO repo calls

	// Create request with malicious content (text/plain) but spoofed header (image/jpeg)
	req, _ := createMultipartRequest(t, "image", "evil.php", "image/jpeg", []byte("<?php echo 'pwned'; ?>"))

	// Add user context
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sessionID := uuid.New()
	ctx := middleware.SetUserContext(req.Context(), userID, "test@example.com", "user", sessionID, false)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	// Act
	handler.Upload(rec, req)

	// Assert
	// We expect validation failure (400 Bad Request) because it's not an image
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	mockRepo.AssertNotCalled(t, "Save")
	mockStorage.AssertNotCalled(t, "Put")
}

func TestImageHandler_Upload_ValidImage(t *testing.T) {
	// Arrange
	mockRepo := new(MockImageRepository)
	mockStorage := new(MockStorage)
	mockJobEnqueuer := new(MockJobEnqueuer)
	mockEventPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()
	loggerPtr := &logger

	// Create command handler with mocks
	uploadHandler := commands.NewUploadImageHandler(
		mockRepo,
		mockStorage,
		mockJobEnqueuer,
		mockEventPublisher,
		loggerPtr,
	)

	// Create image handler
	handler := NewImageHandler(
		uploadHandler,
		nil, nil, nil, nil, nil, nil,
		mockStorage,
		logger,
	)

	testImageID := gallery.NewImageID()
	// Expectations
	mockRepo.On("NextID").Return(testImageID)
	mockStorage.On("Provider").Return("local")

	// Expect Put with detected GIF mime type
	mockStorage.On("Put",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.Anything,
		mock.AnythingOfType("int64"),
		mock.MatchedBy(func(opts storage.PutOptions) bool {
			return opts.ContentType == "image/gif"
		}),
	).Return(nil)

	// Expect Save with detected dimensions (1x1)
	mockRepo.On("Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
		m := img.Metadata()
		return m.MimeType() == "image/gif" && m.Width() == 1 && m.Height() == 1
	})).Return(nil)

	mockJobEnqueuer.On("EnqueueImageProcessing", mock.Anything, testImageID.String()).Return(nil)
	mockEventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

	// Valid 1x1 GIF
	validGif := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x01, 0x44, 0x00, 0x3b,
	}

	// Request with "image/jpeg" header (spoofed, but content is GIF)
	// We expect the handler to detect it's GIF and use GIF
	req, _ := createMultipartRequest(t, "image", "valid.gif", "image/jpeg", validGif)

	// Add user context
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sessionID := uuid.New()
	ctx := middleware.SetUserContext(req.Context(), userID, "test@example.com", "user", sessionID, false)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	// Act
	handler.Upload(rec, req)

	// Assert
	assert.Equal(t, http.StatusCreated, rec.Code)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockJobEnqueuer.AssertExpectations(t)
}
