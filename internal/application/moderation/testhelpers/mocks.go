package testhelpers

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// MockBanRepository is a mock implementation of moderation.BanRepository.
type MockBanRepository struct {
	mock.Mock
}

func (m *MockBanRepository) NextID() moderation.BanID {
	args := m.Called()
	return args.Get(0).(moderation.BanID)
}

func (m *MockBanRepository) FindByID(ctx context.Context, id moderation.BanID) (*moderation.Ban, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*moderation.Ban), args.Error(1)
}

func (m *MockBanRepository) FindByUserID(ctx context.Context, userID identity.UserID) (*moderation.Ban, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*moderation.Ban), args.Error(1)
}

func (m *MockBanRepository) FindActiveBans(ctx context.Context) ([]*moderation.Ban, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*moderation.Ban), args.Error(1)
}

func (m *MockBanRepository) FindExpiredBans(ctx context.Context) ([]*moderation.Ban, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*moderation.Ban), args.Error(1)
}

func (m *MockBanRepository) IsUserBanned(ctx context.Context, userID identity.UserID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockBanRepository) Save(ctx context.Context, ban *moderation.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

// MockReportRepository is a mock implementation of moderation.ReportRepository.
type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) NextID() moderation.ReportID {
	args := m.Called()
	return args.Get(0).(moderation.ReportID)
}

func (m *MockReportRepository) FindByID(ctx context.Context, id moderation.ReportID) (*moderation.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*moderation.Report), args.Error(1)
}

func (m *MockReportRepository) FindPending(ctx context.Context, pagination shared.Pagination) ([]*moderation.Report, int64, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*moderation.Report), args.Get(1).(int64), args.Error(2)
}

func (m *MockReportRepository) FindByImage(ctx context.Context, imageID gallery.ImageID) ([]*moderation.Report, error) {
	args := m.Called(ctx, imageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*moderation.Report), args.Error(1)
}

func (m *MockReportRepository) FindByReporter(ctx context.Context, reporterID identity.UserID, pagination shared.Pagination) ([]*moderation.Report, int64, error) {
	args := m.Called(ctx, reporterID, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*moderation.Report), args.Get(1).(int64), args.Error(2)
}

func (m *MockReportRepository) Save(ctx context.Context, report *moderation.Report) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

// MockReviewRepository is a mock implementation of moderation.ReviewRepository.
type MockReviewRepository struct {
	mock.Mock
}

func (m *MockReviewRepository) NextID() moderation.ReviewID {
	args := m.Called()
	return args.Get(0).(moderation.ReviewID)
}

func (m *MockReviewRepository) FindByID(ctx context.Context, id moderation.ReviewID) (*moderation.Review, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*moderation.Review), args.Error(1)
}

func (m *MockReviewRepository) FindByReport(ctx context.Context, reportID moderation.ReportID) ([]*moderation.Review, error) {
	args := m.Called(ctx, reportID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*moderation.Review), args.Error(1)
}

func (m *MockReviewRepository) FindByReviewer(ctx context.Context, reviewerID identity.UserID, pagination shared.Pagination) ([]*moderation.Review, int64, error) {
	args := m.Called(ctx, reviewerID, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*moderation.Review), args.Get(1).(int64), args.Error(2)
}

func (m *MockReviewRepository) Save(ctx context.Context, review *moderation.Review) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

// MockNSFWScanRepository is a mock implementation of moderation.NSFWScanRepository.
type MockNSFWScanRepository struct {
	mock.Mock
}

func (m *MockNSFWScanRepository) NextID() moderation.NSFWScanID {
	args := m.Called()
	return args.Get(0).(moderation.NSFWScanID)
}

func (m *MockNSFWScanRepository) FindByID(ctx context.Context, id moderation.NSFWScanID) (*moderation.NSFWScan, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*moderation.NSFWScan), args.Error(1)
}

func (m *MockNSFWScanRepository) FindByImageID(ctx context.Context, imageID gallery.ImageID) (*moderation.NSFWScan, error) {
	args := m.Called(ctx, imageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*moderation.NSFWScan), args.Error(1)
}

func (m *MockNSFWScanRepository) FindByImageIDAll(ctx context.Context, imageID gallery.ImageID) ([]*moderation.NSFWScan, error) {
	args := m.Called(ctx, imageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*moderation.NSFWScan), args.Error(1)
}

func (m *MockNSFWScanRepository) FindPending(ctx context.Context, pagination shared.Pagination) ([]*moderation.NSFWScan, int64, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*moderation.NSFWScan), args.Get(1).(int64), args.Error(2)
}

func (m *MockNSFWScanRepository) FindByStatus(ctx context.Context, status moderation.NSFWScanStatus, pagination shared.Pagination) ([]*moderation.NSFWScan, int64, error) {
	args := m.Called(ctx, status, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*moderation.NSFWScan), args.Get(1).(int64), args.Error(2)
}

func (m *MockNSFWScanRepository) FindNSFWImages(ctx context.Context, pagination shared.Pagination) ([]*moderation.NSFWScan, int64, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*moderation.NSFWScan), args.Get(1).(int64), args.Error(2)
}

func (m *MockNSFWScanRepository) HasActiveScan(ctx context.Context, imageID gallery.ImageID) (bool, error) {
	args := m.Called(ctx, imageID)
	return args.Bool(0), args.Error(1)
}

func (m *MockNSFWScanRepository) Save(ctx context.Context, scan *moderation.NSFWScan) error {
	args := m.Called(ctx, scan)
	return args.Error(0)
}

// MockEventPublisher is a mock implementation of moderation.EventPublisher.
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockImageRepository is a mock implementation of gallery.ImageRepository for moderation tests.
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
