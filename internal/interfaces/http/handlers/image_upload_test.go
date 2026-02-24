package handlers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	appgallery "github.com/yegamble/goimg-datalayer/internal/application/gallery"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

type MockStorage struct {
	mock.Mock
}

// Note: This only works because we pass a struct to NewImageHandler that matches both interfaces where necessary

type MockAppStorage struct {
	mock.Mock
}

func (m *MockAppStorage) Put(ctx context.Context, key string, data io.Reader, size int64, opts appgallery.PutOptions) error {
	args := m.Called(ctx, key, data, size, opts)
	return args.Error(0)
}

func (m *MockAppStorage) GetBytes(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockAppStorage) Provider() string {
	return "mock"
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
	return "mock"
}

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

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestImageHandler_Upload_MimeTypeDetection(t *testing.T) {
	mockStorage := new(MockStorage)
	mockAppStorage := new(MockAppStorage)
	mockRepo := new(MockImageRepository)
	mockJobEnqueuer := new(MockJobEnqueuer)
	mockEventPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()

	uploadHandler := commands.NewUploadImageHandler(mockRepo, mockAppStorage, mockJobEnqueuer, mockEventPublisher, &logger)
	imageHandler := NewImageHandler(
		uploadHandler, nil, nil, nil, nil, nil, nil, mockStorage, "", logger,
	)

	userID := identity.NewUserID()
	email := "test@example.com"
	role := "user"
	sessionID := uuid.New()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="image"; filename="fake.jpg"`)
	h.Set("Content-Type", "image/jpeg")

	part, err := writer.CreatePart(h)
	assert.NoError(t, err)
	_, err = part.Write([]byte("This is text, not an image"))
	assert.NoError(t, err)

	err = writer.WriteField("title", "Test Image")
	assert.NoError(t, err)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	ctx := middleware.SetUserContext(req.Context(), userID.UUID(), email, role, sessionID, false, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	mockRepo.On("NextID").Return(gallery.NewImageID())
	// Note: UploadHandler calls mockAppStorage.Put, not mockStorage.Put
	mockAppStorage.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockJobEnqueuer.On("EnqueueImageProcessing", mock.Anything, mock.Anything).Return(nil).Maybe()

	imageHandler.Upload(rec, req)


	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	mockAppStorage.AssertNotCalled(t, "Put")
}
