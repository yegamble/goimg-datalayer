package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestListNSFWFlaggedHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   queries.ListNSFWFlaggedQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListNSFWFlaggedResult, err error)
	}{
		{
			name: "successful list",
			query: queries.ListNSFWFlaggedQuery{
				Page:     1,
				PageSize: 10,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				scanID, _ := moderation.ParseNSFWScanID(testhelpers.ValidImageID) // Reuse ID for simplicity
				imageID := testhelpers.ValidImageIDParsed()
				scan := moderation.NewNSFWScan(scanID, imageID, moderation.ProviderSightEngine)
				_ = scan.MarkScanning()
				_ = scan.Complete(moderation.CategoryNudity, 0.99, moderation.NSFWDetails{})

				scans := []*moderation.NSFWScan{scan}
				suite.NSFWScanRepo.On("FindNSFWImages", mock.Anything, mock.Anything).Return(scans, int64(1), nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListNSFWFlaggedResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, int64(1), result.TotalCount)
				assert.Equal(t, 1, result.Page)
				assert.Equal(t, 10, result.PageSize)
				assert.Equal(t, int64(1), result.TotalPages)
				assert.Len(t, result.Scans, 1)
				assert.Equal(t, "nudity", result.Scans[0].Category)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid pagination",
			query: queries.ListNSFWFlaggedQuery{
				Page:     0,
				PageSize: 10,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid pagination",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListNSFWFlaggedResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid pagination")
			},
		},
		{
			name: "repository error",
			query: queries.ListNSFWFlaggedQuery{
				Page:     1,
				PageSize: 10,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.NSFWScanRepo.On("FindNSFWImages", mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("db error")).Once()
			},
			wantErr: "find NSFW flagged scans",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListNSFWFlaggedResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find NSFW flagged scans")
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := queries.NewListNSFWFlaggedHandler(suite.NSFWScanRepo)
			result, err := handler.Handle(context.Background(), tt.query)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
