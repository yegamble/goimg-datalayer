package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestListActiveBansHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   queries.ListActiveBansQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListActiveBansResult, err error)
	}{
		{
			name:  "successful list with bans",
			query: queries.ListActiveBansQuery{},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				ban1 := testhelpers.ValidBan(t)
				ban2 := testhelpers.ValidTemporaryBan(t, 24*time.Hour)
				bans := []*moderation.Ban{ban1, ban2}

				suite.BanRepo.On("FindActiveBans", mock.Anything).Return(bans, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListActiveBansResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 2, result.TotalCount)
				assert.Len(t, result.Bans, 2)
				assert.Equal(t, testhelpers.ValidBanReason, result.Bans[0].Reason)
				suite.AssertExpectations(t)
			},
		},
		{
			name:  "successful list empty",
			query: queries.ListActiveBansQuery{},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.BanRepo.On("FindActiveBans", mock.Anything).Return([]*moderation.Ban{}, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListActiveBansResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 0, result.TotalCount)
				assert.Empty(t, result.Bans)
				suite.AssertExpectations(t)
			},
		},
		{
			name:  "repository error",
			query: queries.ListActiveBansQuery{},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.BanRepo.On("FindActiveBans", mock.Anything).Return(nil, errors.New("db error")).Once()
			},
			wantErr: "find active bans",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListActiveBansResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find active bans")
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := queries.NewListActiveBansHandler(suite.BanRepo, &suite.Logger)
			result, err := handler.Handle(context.Background(), tt.query)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
