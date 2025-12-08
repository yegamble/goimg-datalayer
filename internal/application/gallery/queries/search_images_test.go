//nolint:testpackage // White-box testing required for internal mocks
package queries

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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
		if err := args.Error(1); err != nil {
			return nil, fmt.Errorf("mock FindByID: %w", err)
		}
		return nil, errors.New("mock: no image found")
	}
	if err := args.Error(1); err != nil {
		return args.Get(0).(*gallery.Image), fmt.Errorf("mock FindByID: %w", err)
	}
	return args.Get(0).(*gallery.Image), nil
}

func (m *MockImageRepository) FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	args := m.Called(ctx, ownerID, pagination)
	count := args.Get(1).(int64)
	if args.Get(0) == nil {
		if err := args.Error(2); err != nil {
			return nil, count, fmt.Errorf("mock FindByOwner: %w", err)
		}
		return nil, count, nil
	}
	if err := args.Error(2); err != nil {
		return args.Get(0).([]*gallery.Image), count, fmt.Errorf("mock FindByOwner: %w", err)
	}
	return args.Get(0).([]*gallery.Image), count, nil
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

func TestSearchImagesHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockImages := new(MockImageRepository)
	handler := NewSearchImagesHandler(mockImages)

	userID := identity.NewUserID()
	metadata, _ := gallery.NewImageMetadata(
		"Test Image",
		"Test description",
		"test.jpg",
		"image/jpeg",
		800,
		600,
		1024,
		"path/to/image.jpg",
		"local",
	)

	now := time.Now().UTC()
	image := gallery.ReconstructImage(
		gallery.NewImageID(),
		userID,
		metadata,
		gallery.VisibilityPublic,
		gallery.StatusActive,
		[]gallery.ImageVariant{},
		[]gallery.Tag{},
		100,
		50,
		10,
		now,
		now,
	)

	query := SearchImagesQuery{
		Query:      "test",
		Tags:       []string{},
		OwnerID:    "",
		Visibility: "public",
		SortBy:     "relevance",
		Page:       1,
		PerPage:    20,
	}

	// Mock expectations
	mockImages.On("Search", mock.Anything, mock.MatchedBy(func(params gallery.SearchParams) bool {
		return params.Query == "test" &&
			len(params.Tags) == 0 &&
			params.OwnerID == nil &&
			params.Visibility != nil &&
			*params.Visibility == gallery.VisibilityPublic &&
			params.SortBy == gallery.SearchSortByRelevance
	})).Return([]*gallery.Image{image}, int64(1), nil)

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.TotalCount)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PerPage)
	assert.Equal(t, 1, result.TotalPages)
	assert.Equal(t, "test", result.Query)
	assert.Len(t, result.Images, 1)

	mockImages.AssertExpectations(t)
}

func TestSearchImagesHandler_Handle_WithTags(t *testing.T) {
	t.Parallel()

	// Arrange
	mockImages := new(MockImageRepository)
	handler := NewSearchImagesHandler(mockImages)

	query := SearchImagesQuery{
		Query:      "landscape",
		Tags:       []string{"nature", "mountains"},
		OwnerID:    "",
		Visibility: "public",
		SortBy:     "created_at",
		Page:       1,
		PerPage:    10,
	}

	// Mock expectations
	mockImages.On("Search", mock.Anything, mock.MatchedBy(func(params gallery.SearchParams) bool {
		return params.Query == "landscape" &&
			len(params.Tags) == 2 &&
			params.SortBy == gallery.SearchSortByCreatedAt
	})).Return([]*gallery.Image{}, int64(0), nil)

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), result.TotalCount)
	assert.Empty(t, result.Images)

	mockImages.AssertExpectations(t)
}

func TestSearchImagesHandler_Handle_InvalidOwnerID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockImages := new(MockImageRepository)
	handler := NewSearchImagesHandler(mockImages)

	query := SearchImagesQuery{
		Query:      "test",
		Tags:       []string{},
		OwnerID:    "invalid-uuid",
		Visibility: "public",
		SortBy:     "relevance",
		Page:       1,
		PerPage:    20,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid owner user id")

	mockImages.AssertNotCalled(t, "Search")
}

func TestSearchImagesHandler_Handle_InvalidVisibility(t *testing.T) {
	t.Parallel()

	// Arrange
	mockImages := new(MockImageRepository)
	handler := NewSearchImagesHandler(mockImages)

	query := SearchImagesQuery{
		Query:      "test",
		Tags:       []string{},
		OwnerID:    "",
		Visibility: "invalid",
		SortBy:     "relevance",
		Page:       1,
		PerPage:    20,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid visibility")

	mockImages.AssertNotCalled(t, "Search")
}

func TestSearchImagesHandler_Handle_InvalidTag(t *testing.T) {
	t.Parallel()

	// Arrange
	mockImages := new(MockImageRepository)
	handler := NewSearchImagesHandler(mockImages)

	query := SearchImagesQuery{
		Query:      "test",
		Tags:       []string{"valid", ""}, // Empty tag name is invalid
		OwnerID:    "",
		Visibility: "public",
		SortBy:     "relevance",
		Page:       1,
		PerPage:    20,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid tag")

	mockImages.AssertNotCalled(t, "Search")
}

func TestSearchImagesHandler_Handle_InvalidSortBy(t *testing.T) {
	t.Parallel()

	// Arrange
	mockImages := new(MockImageRepository)
	handler := NewSearchImagesHandler(mockImages)

	query := SearchImagesQuery{
		Query:      "test",
		Tags:       []string{},
		OwnerID:    "",
		Visibility: "public",
		SortBy:     "invalid_sort",
		Page:       1,
		PerPage:    20,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid sort parameter")

	mockImages.AssertNotCalled(t, "Search")
}

func TestSearchImagesHandler_Handle_SearchError(t *testing.T) {
	t.Parallel()

	// Arrange
	mockImages := new(MockImageRepository)
	handler := NewSearchImagesHandler(mockImages)

	query := SearchImagesQuery{
		Query:      "test",
		Tags:       []string{},
		OwnerID:    "",
		Visibility: "public",
		SortBy:     "relevance",
		Page:       1,
		PerPage:    20,
	}

	// Mock expectations
	mockImages.On("Search", mock.Anything, mock.Anything).
		Return(nil, int64(0), errors.New("database error"))

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "search images")

	mockImages.AssertExpectations(t)
}

func TestSearchImagesHandler_Handle_DefaultPagination(t *testing.T) {
	t.Parallel()

	// Arrange
	mockImages := new(MockImageRepository)
	handler := NewSearchImagesHandler(mockImages)

	query := SearchImagesQuery{
		Query:      "test",
		Tags:       []string{},
		OwnerID:    "",
		Visibility: "",
		SortBy:     "",
		Page:       0, // Invalid page (defaults to 1)
		PerPage:    0, // Invalid per_page (defaults to 20)
	}

	// Mock expectations
	mockImages.On("Search", mock.Anything, mock.Anything).
		Return([]*gallery.Image{}, int64(0), nil)

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Page)     // Default page
	assert.Equal(t, 20, result.PerPage) // Default per_page

	mockImages.AssertExpectations(t)
}

func TestSearchImagesHandler_Handle_SortByViewCount(t *testing.T) {
	t.Parallel()

	// Arrange
	mockImages := new(MockImageRepository)
	handler := NewSearchImagesHandler(mockImages)

	query := SearchImagesQuery{
		Query:      "popular",
		Tags:       []string{},
		OwnerID:    "",
		Visibility: "public",
		SortBy:     "view_count",
		Page:       1,
		PerPage:    10,
	}

	// Mock expectations
	mockImages.On("Search", mock.Anything, mock.MatchedBy(func(params gallery.SearchParams) bool {
		return params.SortBy == gallery.SearchSortByViewCount
	})).Return([]*gallery.Image{}, int64(0), nil)

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	mockImages.AssertExpectations(t)
}
