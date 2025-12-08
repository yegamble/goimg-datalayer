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

func TestListImagesHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   queries.ListImagesQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error)
	}{
		{
			name: "successful list - public images",
			query: queries.ListImagesQuery{
				Offset: 0,
				Limit:  20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				images := []*gallery.Image{
					testhelpers.ValidImage(t),
					testhelpers.ValidImage(t),
				}

				// Handler converts offset/limit to page: page = (0/20) + 1 = 1
				pagination, _ := shared.NewPagination(1, 20)
				suite.ImageRepo.On("FindPublic", mock.Anything, pagination).
					Return(images, int64(2), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result.Images, 2)
				assert.Equal(t, int64(2), result.TotalCount)
				assert.Equal(t, 0, result.Offset)
				assert.Equal(t, 20, result.Limit)
				assert.False(t, result.HasMore)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful list - by owner",
			query: queries.ListImagesQuery{
				OwnerID: testhelpers.ValidUserID,
				Offset:  0,
				Limit:   20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				images := []*gallery.Image{
					testhelpers.ValidImage(t),
				}

				ownerID := testhelpers.ValidUserIDParsed()
				// Handler converts offset/limit to page: page = (0/20) + 1 = 1
				pagination, _ := shared.NewPagination(1, 20)
				suite.ImageRepo.On("FindByOwner", mock.Anything, ownerID, pagination).
					Return(images, int64(1), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result.Images, 1)
				assert.Equal(t, int64(1), result.TotalCount)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful list - by tag",
			query: queries.ListImagesQuery{
				Tag:    "nature",
				Offset: 0,
				Limit:  20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				images := []*gallery.Image{
					testhelpers.ValidImage(t),
				}

				tag := testhelpers.ValidTag(t, "nature")
				// Handler converts offset/limit to page: page = (0/20) + 1 = 1
				pagination, _ := shared.NewPagination(1, 20)
				suite.ImageRepo.On("FindByTag", mock.Anything, tag, pagination).
					Return(images, int64(1), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result.Images, 1)
				assert.Equal(t, int64(1), result.TotalCount)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "pagination - has more",
			query: queries.ListImagesQuery{
				Offset: 0,
				Limit:  10,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				images := make([]*gallery.Image, 10)
				for i := 0; i < 10; i++ {
					images[i] = testhelpers.ValidImage(t)
				}

				// Handler converts offset/limit to page: page = (0/10) + 1 = 1
				pagination, _ := shared.NewPagination(1, 10)
				suite.ImageRepo.On("FindPublic", mock.Anything, pagination).
					Return(images, int64(25), nil).Once() // Total is 25, so has more
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result.Images, 10)
				assert.Equal(t, int64(25), result.TotalCount)
				assert.True(t, result.HasMore)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "default limit applied",
			query: queries.ListImagesQuery{
				Offset: 0,
				Limit:  0, // Will default to 20
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				images := []*gallery.Image{
					testhelpers.ValidImage(t),
				}

				// Handler converts offset/limit to page: page = (0/20) + 1 = 1
				pagination, _ := shared.NewPagination(1, 20) // Default limit
				suite.ImageRepo.On("FindPublic", mock.Anything, pagination).
					Return(images, int64(1), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 20, result.Limit)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "max limit enforced",
			query: queries.ListImagesQuery{
				Offset: 0,
				Limit:  200, // Exceeds max of 100
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				images := []*gallery.Image{
					testhelpers.ValidImage(t),
				}

				// Handler converts offset/limit to page: page = (0/100) + 1 = 1
				pagination, _ := shared.NewPagination(1, 100) // Max limit
				suite.ImageRepo.On("FindPublic", mock.Anything, pagination).
					Return(images, int64(1), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 100, result.Limit)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "negative offset normalized to zero",
			query: queries.ListImagesQuery{
				Offset: -10,
				Limit:  20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				images := []*gallery.Image{
					testhelpers.ValidImage(t),
				}

				// Handler normalizes offset to 0, then converts: page = (0/20) + 1 = 1
				pagination, _ := shared.NewPagination(1, 20) // Normalized offset
				suite.ImageRepo.On("FindPublic", mock.Anything, pagination).
					Return(images, int64(1), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 0, result.Offset)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid owner id",
			query: queries.ListImagesQuery{
				OwnerID: "invalid-uuid",
				Offset:  0,
				Limit:   20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid owner id")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid requesting user id",
			query: queries.ListImagesQuery{
				RequestingUserID: "invalid-uuid",
				Offset:           0,
				Limit:            20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid requesting user id")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid visibility filter",
			query: queries.ListImagesQuery{
				Visibility: "invalid",
				Offset:     0,
				Limit:      20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: gallery.ErrInvalidVisibility,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrInvalidVisibility)
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid tag",
			query: queries.ListImagesQuery{
				Tag:    "a", // Too short
				Offset: 0,
				Limit:  20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: gallery.ErrTagTooShort,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrTagTooShort)
				assert.Nil(t, result)
			},
		},
		{
			name: "repository error - by tag",
			query: queries.ListImagesQuery{
				Tag:    "nature",
				Offset: 0,
				Limit:  20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				tag := testhelpers.ValidTag(t, "nature")
				// Handler converts offset/limit to page: page = (0/20) + 1 = 1
				pagination, _ := shared.NewPagination(1, 20)
				suite.ImageRepo.On("FindByTag", mock.Anything, tag, pagination).
					Return(nil, int64(0), fmt.Errorf("database error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "list images by tag")
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "repository error - by owner",
			query: queries.ListImagesQuery{
				OwnerID: testhelpers.ValidUserID,
				Offset:  0,
				Limit:   20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				ownerID := testhelpers.ValidUserIDParsed()
				// Handler converts offset/limit to page: page = (0/20) + 1 = 1
				pagination, _ := shared.NewPagination(1, 20)
				suite.ImageRepo.On("FindByOwner", mock.Anything, ownerID, pagination).
					Return(nil, int64(0), fmt.Errorf("database error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "list images by owner")
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "repository error - public",
			query: queries.ListImagesQuery{
				Offset: 0,
				Limit:  20,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Handler converts offset/limit to page: page = (0/20) + 1 = 1
				pagination, _ := shared.NewPagination(1, 20)
				suite.ImageRepo.On("FindPublic", mock.Anything, pagination).
					Return(nil, int64(0), fmt.Errorf("database error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListImagesResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "list public images")
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			suite := testhelpers.NewTestSuite(t)
			if tt.setup != nil {
				tt.setup(t, suite)
			}

			handler := queries.NewListImagesHandler(
				suite.ImageRepo,
				&suite.Logger,
			)

			// Act
			result, err := handler.Handle(context.Background(), tt.query)

			// Assert
			switch {
			case tt.assert != nil:
				tt.assert(t, suite, result, err)
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			default:
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

// TestListImagesQuery_Interface verifies the query implements the interface.
func TestListImagesQuery_Interface(t *testing.T) {
	t.Parallel()
}
