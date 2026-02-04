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

func TestListNSFWScansByImageHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   queries.ListNSFWScansByImageQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListNSFWScansByImageResult, err error)
	}{
		{
			name: "successful list",
			query: queries.ListNSFWScansByImageQuery{
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				scanID, _ := moderation.ParseNSFWScanID(testhelpers.ValidImageID)
				scan := moderation.NewNSFWScan(scanID, imageID, moderation.ProviderSightEngine)

				scans := []*moderation.NSFWScan{scan}
				suite.NSFWScanRepo.On("FindByImageIDAll", mock.Anything, imageID).Return(scans, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListNSFWScansByImageResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 1, result.TotalCount)
				assert.Len(t, result.Scans, 1)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid image id",
			query: queries.ListNSFWScansByImageQuery{
				ImageID: "invalid-uuid",
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid image id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListNSFWScansByImageResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid image id")
			},
		},
		{
			name: "repository error",
			query: queries.ListNSFWScansByImageQuery{
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.NSFWScanRepo.On("FindByImageIDAll", mock.Anything, imageID).Return(nil, errors.New("db error")).Once()
			},
			wantErr: "find scans by image id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListNSFWScansByImageResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find scans by image id")
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := queries.NewListNSFWScansByImageHandler(suite.NSFWScanRepo)
			result, err := handler.Handle(context.Background(), tt.query)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
