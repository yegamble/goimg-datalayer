package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
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

func (m *MockImageRepository) FindByIDs(ctx context.Context, ids []gallery.ImageID) ([]*gallery.Image, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gallery.Image), args.Error(1)
}

func (m *MockImageRepository) FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination, visibility *gallery.Visibility) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, ownerID, pagination, visibility)
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

func (m *MockImageRepository) ExistsByID(ctx context.Context, id gallery.ImageID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func TestImageHandler_Search_CommaSeparatedTags(t *testing.T) {
	// Arrange
	mockRepo := new(MockImageRepository)
	searchHandler := queries.NewSearchImagesHandler(mockRepo)

	imageHandler := NewImageHandler(
		nil, nil, nil, nil, nil, nil,
		searchHandler,
		nil, // StorageProvider not needed for Search
		zerolog.Nop(),
	)

	// We expect the repository Search method to be called with both "tag1" and "tag2"
	mockRepo.On("Search", mock.Anything, mock.MatchedBy(func(params gallery.SearchParams) bool {
		if len(params.Tags) != 2 {
			return false
		}
		// Checking that both tags are present
		return params.Tags[0].String() == "tag1" && params.Tags[1].String() == "tag2"
	})).Return([]*gallery.Image{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&tags=tag1,tag2", nil)
	rec := httptest.NewRecorder()

	// Act
	imageHandler.Search(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	mockRepo.AssertExpectations(t)
}

func TestImageHandler_Search_SingleTag(t *testing.T) {
	// Arrange
	mockRepo := new(MockImageRepository)
	searchHandler := queries.NewSearchImagesHandler(mockRepo)

	imageHandler := NewImageHandler(
		nil, nil, nil, nil, nil, nil,
		searchHandler,
		nil,
		zerolog.Nop(),
	)

	mockRepo.On("Search", mock.Anything, mock.MatchedBy(func(params gallery.SearchParams) bool {
		if len(params.Tags) != 1 {
			return false
		}
		return params.Tags[0].String() == "tag1"
	})).Return([]*gallery.Image{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&tags=tag1", nil)
	rec := httptest.NewRecorder()

	// Act
	imageHandler.Search(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	mockRepo.AssertExpectations(t)
}

func TestImageHandler_Search_TagsWithSpaces(t *testing.T) {
	// Arrange
	mockRepo := new(MockImageRepository)
	searchHandler := queries.NewSearchImagesHandler(mockRepo)

	imageHandler := NewImageHandler(
		nil, nil, nil, nil, nil, nil,
		searchHandler,
		nil,
		zerolog.Nop(),
	)

	// " tag1 , tag2 " -> "tag1", "tag2"
	mockRepo.On("Search", mock.Anything, mock.MatchedBy(func(params gallery.SearchParams) bool {
		if len(params.Tags) != 2 {
			return false
		}
		return params.Tags[0].String() == "tag1" && params.Tags[1].String() == "tag2"
	})).Return([]*gallery.Image{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&tags=%20tag1%20,%20tag2%20", nil)
	rec := httptest.NewRecorder()

	// Act
	imageHandler.Search(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	mockRepo.AssertExpectations(t)
}
