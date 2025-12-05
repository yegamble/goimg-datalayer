package queries_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestListImageCommentsHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("successful list - oldest first (default)", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		imageID := testhelpers.ValidImageIDParsed()
		comments := []*gallery.Comment{
			testhelpers.ValidComment(t),
			testhelpers.ValidComment(t),
		}

		pagination, _ := shared.NewPagination(1, 20)
		mockCommentRepo.On("FindByImage", mock.Anything, imageID, pagination).
			Return(comments, int64(2), nil).Once()

		query := queries.ListImageCommentsQuery{
			ImageID:   testhelpers.ValidImageID,
			Page:      1,
			PerPage:   20,
			SortOrder: "oldest",
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Comments, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 1, result.Pagination.Page())
		assert.Equal(t, 20, result.Pagination.PerPage())
		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("successful list - newest first (reversed)", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		imageID := testhelpers.ValidImageIDParsed()

		// Create comments with different content to verify reversal
		comment1 := testhelpers.ValidComment(t)
		comment2 := testhelpers.ValidComment(t)
		comments := []*gallery.Comment{comment1, comment2}

		pagination, _ := shared.NewPagination(1, 20)
		mockCommentRepo.On("FindByImage", mock.Anything, imageID, pagination).
			Return(comments, int64(2), nil).Once()

		query := queries.ListImageCommentsQuery{
			ImageID:   testhelpers.ValidImageID,
			Page:      1,
			PerPage:   20,
			SortOrder: "newest",
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Comments, 2)
		// Verify order is reversed
		assert.Equal(t, comment2.ID(), result.Comments[0].ID())
		assert.Equal(t, comment1.ID(), result.Comments[1].ID())
		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("successful list - empty result", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		imageID := testhelpers.ValidImageIDParsed()
		pagination, _ := shared.NewPagination(1, 20)
		mockCommentRepo.On("FindByImage", mock.Anything, imageID, pagination).
			Return([]*gallery.Comment{}, int64(0), nil).Once()

		query := queries.ListImageCommentsQuery{
			ImageID: testhelpers.ValidImageID,
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Comments, 0)
		assert.Equal(t, int64(0), result.Total)
		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("pagination - multiple pages", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		imageID := testhelpers.ValidImageIDParsed()
		comments := make([]*gallery.Comment, 10)
		for i := 0; i < 10; i++ {
			comments[i] = testhelpers.ValidComment(t)
		}

		pagination, _ := shared.NewPagination(1, 10)
		mockCommentRepo.On("FindByImage", mock.Anything, imageID, pagination).
			Return(comments, int64(45), nil).Once()

		query := queries.ListImageCommentsQuery{
			ImageID: testhelpers.ValidImageID,
			Page:    1,
			PerPage: 10,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Comments, 10)
		assert.Equal(t, int64(45), result.Total)
		assert.Equal(t, 5, result.Pagination.TotalPages()) // 45 / 10 = 5 pages
		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("invalid image id", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		query := queries.ListImageCommentsQuery{
			ImageID: "invalid-uuid",
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid image id")
		assert.Nil(t, result)
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		imageID := testhelpers.ValidImageIDParsed()
		pagination, _ := shared.NewPagination(1, 20)
		mockCommentRepo.On("FindByImage", mock.Anything, imageID, pagination).
			Return(nil, int64(0), fmt.Errorf("database error")).Once()

		query := queries.ListImageCommentsQuery{
			ImageID: testhelpers.ValidImageID,
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find comments by image")
		assert.Nil(t, result)
		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("default pagination applied", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		imageID := testhelpers.ValidImageIDParsed()
		comments := []*gallery.Comment{testhelpers.ValidComment(t)}

		defaultPagination := shared.DefaultPagination()
		mockCommentRepo.On("FindByImage", mock.Anything, imageID, defaultPagination).
			Return(comments, int64(1), nil).Once()

		query := queries.ListImageCommentsQuery{
			ImageID: testhelpers.ValidImageID,
			Page:    0, // Invalid, will use default
			PerPage: 0, // Invalid, will use default
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultPagination.Page(), result.Pagination.Page())
		assert.Equal(t, defaultPagination.PerPage(), result.Pagination.PerPage())
		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("sort order - empty string defaults to oldest", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		imageID := testhelpers.ValidImageIDParsed()
		comment1 := testhelpers.ValidComment(t)
		comment2 := testhelpers.ValidComment(t)
		comments := []*gallery.Comment{comment1, comment2}

		pagination, _ := shared.NewPagination(1, 20)
		mockCommentRepo.On("FindByImage", mock.Anything, imageID, pagination).
			Return(comments, int64(2), nil).Once()

		query := queries.ListImageCommentsQuery{
			ImageID:   testhelpers.ValidImageID,
			Page:      1,
			PerPage:   20,
			SortOrder: "", // Empty defaults to oldest
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		// Order should not be reversed (oldest first)
		assert.Equal(t, comment1.ID(), result.Comments[0].ID())
		assert.Equal(t, comment2.ID(), result.Comments[1].ID())
		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("newest sort with empty result does not panic", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		imageID := testhelpers.ValidImageIDParsed()
		pagination, _ := shared.NewPagination(1, 20)
		mockCommentRepo.On("FindByImage", mock.Anything, imageID, pagination).
			Return([]*gallery.Comment{}, int64(0), nil).Once()

		query := queries.ListImageCommentsQuery{
			ImageID:   testhelpers.ValidImageID,
			Page:      1,
			PerPage:   20,
			SortOrder: "newest",
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Comments, 0)
		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("newest sort with single comment does not panic", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockCommentRepo := new(testhelpers.MockCommentRepository)
		handler := queries.NewListImageCommentsHandler(mockCommentRepo)

		imageID := testhelpers.ValidImageIDParsed()
		comment := testhelpers.ValidComment(t)
		comments := []*gallery.Comment{comment}

		pagination, _ := shared.NewPagination(1, 20)
		mockCommentRepo.On("FindByImage", mock.Anything, imageID, pagination).
			Return(comments, int64(1), nil).Once()

		query := queries.ListImageCommentsQuery{
			ImageID:   testhelpers.ValidImageID,
			Page:      1,
			PerPage:   20,
			SortOrder: "newest",
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Comments, 1)
		assert.Equal(t, comment.ID(), result.Comments[0].ID())
		mockCommentRepo.AssertExpectations(t)
	})
}
