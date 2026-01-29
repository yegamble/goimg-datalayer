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

	"github.com/yegamble/goimg-datalayer/internal/application/notification"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	domainnotification "github.com/yegamble/goimg-datalayer/internal/domain/notification"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/email"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/clamav"
)

// Mocks

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

func (m *MockScanner) ScanReader(ctx context.Context, r io.Reader, size int64) (*clamav.ScanResult, error) {
	// Not used in this test
	return nil, nil
}

func (m *MockScanner) Stats(ctx context.Context) (string, error) {
	return "", nil
}

func (m *MockScanner) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockScanner) Version(ctx context.Context) (string, error) {
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

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
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

func (m *MockImageRepository) FindByIDs(ctx context.Context, ids []gallery.ImageID) ([]*gallery.Image, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gallery.Image), args.Error(1)
}

func (m *MockImageRepository) FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (m *MockImageRepository) FindPublic(ctx context.Context, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (m *MockImageRepository) FindByTag(ctx context.Context, tag gallery.Tag, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (m *MockImageRepository) FindByStatus(ctx context.Context, status gallery.ImageStatus, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (m *MockImageRepository) Search(ctx context.Context, params gallery.SearchParams) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
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

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) NextID() identity.UserID {
	return identity.NewUserID()
}
func (m *MockUserRepository) FindByID(ctx context.Context, id identity.UserID) (*identity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}
func (m *MockUserRepository) FindByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
	return nil, nil
}
func (m *MockUserRepository) FindByUsername(ctx context.Context, username identity.Username) (*identity.User, error) {
	return nil, nil
}
func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *MockUserRepository) Delete(ctx context.Context, id identity.UserID) error {
	return nil
}
func (m *MockUserRepository) ExistsByID(ctx context.Context, id identity.UserID) (bool, error) {
	return false, nil
}
func (m *MockUserRepository) FindExpiredGuests(ctx context.Context, asOf time.Time, limit int) ([]*identity.User, error) {
	return nil, nil
}

// Manually define domain/notification interfaces if needed or reuse existing mock
type MockNotificationRepository struct {
	mock.Mock
}

// Implement notification.NotificationRepository interface methods
func (m *MockNotificationRepository) NextID() domainnotification.NotificationID {
	return domainnotification.NewNotificationID()
}
func (m *MockNotificationRepository) FindByID(ctx context.Context, id domainnotification.NotificationID) (*domainnotification.Notification, error) {
	return nil, nil
}
func (m *MockNotificationRepository) FindByRecipient(ctx context.Context, recipientID identity.UserID, limit, offset int) ([]*domainnotification.Notification, error) {
	return nil, nil
}
func (m *MockNotificationRepository) FindUnreadByRecipient(ctx context.Context, recipientID identity.UserID) ([]*domainnotification.Notification, error) {
	return nil, nil
}
func (m *MockNotificationRepository) GetUnreadCount(ctx context.Context, recipientID identity.UserID) (int64, error) {
	return 0, nil
}
func (m *MockNotificationRepository) CountUnread(ctx context.Context, recipientID identity.UserID) (int, error) {
	return 0, nil
}
func (m *MockNotificationRepository) Save(ctx context.Context, notification *domainnotification.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}
func (m *MockNotificationRepository) MarkAsRead(ctx context.Context, id domainnotification.NotificationID) error {
	return nil
}
func (m *MockNotificationRepository) MarkAllRead(ctx context.Context, recipientID identity.UserID) error {
	return nil
}
func (m *MockNotificationRepository) MarkAllAsRead(ctx context.Context, recipientID identity.UserID) error {
	return nil
}
func (m *MockNotificationRepository) Delete(ctx context.Context, id domainnotification.NotificationID) error {
	return nil
}

func (m *MockNotificationRepository) DeleteOlderThan(ctx context.Context, threshold time.Time) error {
	return nil
}

func TestImageScanHandler_ProcessTask_MalwareDetected(t *testing.T) {
	// Arrange
	mockScanner := new(MockScanner)
	mockStorage := new(MockStorage)
	mockImages := new(MockImageRepository)
	mockUsers := new(MockUserRepository)
	mockNotificationsRepo := new(MockNotificationRepository)
	logger := zerolog.Nop()

	// Create disabled SMTP sender for testing
	smtpCfg := email.Config{Enabled: false}
	smtpSender, _ := email.NewSMTPSender(smtpCfg, logger)

	// Create notification service
	notifService := notification.NewNotificationService(mockNotificationsRepo, mockUsers, smtpSender, logger)

	handler := NewImageScanHandler(
		mockScanner,
		mockStorage,
		mockImages,
		mockUsers,
		notifService,
		logger,
	)

	// Test data
	imageID := gallery.NewImageID()
	userID := identity.NewUserID()
	storageKey := "test/image.jpg"
	filename := "test.jpg"
	fileData := []byte("fake-image-data")

	payload := ImageScanPayload{
		ImageID:          imageID.String(),
		StorageKey:       storageKey,
		OriginalFilename: filename,
		OwnerID:          userID.String(),
		EnqueuedAt:       time.Now(),
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeImageScan, payloadBytes)

	// Mock objects
	emailVal, _ := identity.NewEmail("user@example.com")
	usernameVal, _ := identity.NewUsername("user")
	passwordVal, _ := identity.NewPasswordHash("pass")
	user := identity.ReconstructUser(
		userID, emailVal, usernameVal, passwordVal,
		identity.RoleUser, identity.StatusActive, "User", "", 0,
		time.Now(), time.Now(), identity.UserTypeRegistered, nil, nil,
	)

	metadata, _ := gallery.NewImageMetadata("Title", "Desc", filename, "image/jpeg", 100, 100, 100, storageKey, "local")
	image := gallery.ReconstructImage(
		imageID, userID, metadata, gallery.VisibilityPrivate,
		gallery.StatusProcessing, gallery.ScanStatusPending,
		nil, nil, nil, 0, 0, 0, time.Now(), time.Now(),
	)

	// Expectations
	mockScanner.On("Ping", mock.Anything).Return(nil)
	mockStorage.On("Get", mock.Anything, storageKey).Return(fileData, nil)

	// Simulate malware detected
	scanResult := &clamav.ScanResult{
		Infected:  true,
		Virus:     "EICAR-Test-Signature",
		ScannedAt: time.Now(),
	}
	mockScanner.On("Scan", mock.Anything, fileData).Return(scanResult, nil)

	// Expect storage deletion
	mockStorage.On("Delete", mock.Anything, storageKey).Return(nil)

	// Expect image status update
	mockImages.On("FindByID", mock.Anything, imageID).Return(image, nil)
	mockImages.On("Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
		return img.ScanStatus() == gallery.ScanStatusInfected && img.Status() == gallery.StatusDeleted
	})).Return(nil)

	// Expect user counter increment
	mockUsers.On("FindByID", mock.Anything, userID).Return(user, nil) // Called by both handler and notification service
	mockUsers.On("Save", mock.Anything, mock.MatchedBy(func(u *identity.User) bool {
		return u.InfectedFileCount() == 1
	})).Return(nil)

	// Expect notification creation
	mockNotificationsRepo.On("Save", mock.Anything, mock.MatchedBy(func(n *domainnotification.Notification) bool {
		return n.Type() == domainnotification.TypeMalwareDetected
	})).Return(nil)

	// Act
	err := handler.ProcessTask(context.Background(), task)

	// Assert
	require.NoError(t, err)

	mockScanner.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockImages.AssertExpectations(t)
	mockUsers.AssertExpectations(t)
	mockNotificationsRepo.AssertExpectations(t)
}

func TestImageScanHandler_ProcessTask_Clean(t *testing.T) {
	// Arrange
	mockScanner := new(MockScanner)
	mockStorage := new(MockStorage)
	mockImages := new(MockImageRepository)
	mockUsers := new(MockUserRepository)
	mockNotificationsRepo := new(MockNotificationRepository)
	logger := zerolog.Nop()

	// Create disabled SMTP sender for testing
	smtpCfg := email.Config{Enabled: false}
	smtpSender, _ := email.NewSMTPSender(smtpCfg, logger)

	// Create notification service
	notifService := notification.NewNotificationService(mockNotificationsRepo, mockUsers, smtpSender, logger)

	handler := NewImageScanHandler(
		mockScanner,
		mockStorage,
		mockImages,
		mockUsers,
		notifService,
		logger,
	)

	// Test data
	imageID := gallery.NewImageID()
	userID := identity.NewUserID()
	storageKey := "test/image.jpg"
	filename := "test.jpg"
	fileData := []byte("clean-image-data")

	payload := ImageScanPayload{
		ImageID:          imageID.String(),
		StorageKey:       storageKey,
		OriginalFilename: filename,
		OwnerID:          userID.String(),
		EnqueuedAt:       time.Now(),
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeImageScan, payloadBytes)

	// Mock objects
	metadata, _ := gallery.NewImageMetadata("Title", "Desc", filename, "image/jpeg", 100, 100, 100, storageKey, "local")
	image := gallery.ReconstructImage(
		imageID, userID, metadata, gallery.VisibilityPrivate,
		gallery.StatusProcessing, gallery.ScanStatusPending,
		nil, nil, nil, 0, 0, 0, time.Now(), time.Now(),
	)

	// Expectations
	mockScanner.On("Ping", mock.Anything).Return(nil)
	mockStorage.On("Get", mock.Anything, storageKey).Return(fileData, nil)

	// Simulate clean scan
	scanResult := &clamav.ScanResult{
		Clean:     true,
		ScannedAt: time.Now(),
	}
	mockScanner.On("Scan", mock.Anything, fileData).Return(scanResult, nil)

	// Expect image retrieval
	mockImages.On("FindByID", mock.Anything, imageID).Return(image, nil)

	// Expect image update with Clean status
	mockImages.On("Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
		return img.ScanStatus() == gallery.ScanStatusClean
	})).Return(nil)

	// Act
	err := handler.ProcessTask(context.Background(), task)

	// Assert
	require.NoError(t, err)

	mockScanner.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockImages.AssertExpectations(t)
}
