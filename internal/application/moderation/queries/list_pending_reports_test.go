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

func TestListPendingReportsHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   queries.ListPendingReportsQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListPendingReportsResult, err error)
	}{
		{
			name: "successful retrieval with results",
			query: queries.ListPendingReportsQuery{
				Page:    1,
				PerPage: 10,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				report1 := testhelpers.ValidReport(t)
				report2 := testhelpers.ValidReport(t)
				reports := []*moderation.Report{report1, report2}

				suite.ReportRepo.On("FindPending", mock.Anything, mock.Anything).Return(reports, int64(2), nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListPendingReportsResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result.Reports, 2)
				assert.Equal(t, int64(2), result.TotalCount)
				assert.Equal(t, 1, result.Page)
				assert.Equal(t, 10, result.PerPage)
				assert.Equal(t, 1, result.TotalPages)
				assert.False(t, result.HasNext)
				assert.False(t, result.HasPrev)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "empty result set",
			query: queries.ListPendingReportsQuery{
				Page:    1,
				PerPage: 10,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ReportRepo.On("FindPending", mock.Anything, mock.Anything).Return([]*moderation.Report{}, int64(0), nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListPendingReportsResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result.Reports, 0)
				assert.Equal(t, int64(0), result.TotalCount)
				assert.Equal(t, 0, result.TotalPages)
				assert.False(t, result.HasNext)
				assert.False(t, result.HasPrev)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "pagination with multiple pages",
			query: queries.ListPendingReportsQuery{
				Page:    2,
				PerPage: 10,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				report := testhelpers.ValidReport(t)
				reports := []*moderation.Report{report}

				suite.ReportRepo.On("FindPending", mock.Anything, mock.Anything).Return(reports, int64(25), nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListPendingReportsResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result.Reports, 1)
				assert.Equal(t, int64(25), result.TotalCount)
				assert.Equal(t, 2, result.Page)
				assert.Equal(t, 3, result.TotalPages)
				assert.True(t, result.HasNext)
				assert.True(t, result.HasPrev)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "default page when negative",
			query: queries.ListPendingReportsQuery{
				Page:    -1,
				PerPage: 10,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ReportRepo.On("FindPending", mock.Anything, mock.Anything).Return([]*moderation.Report{}, int64(0), nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListPendingReportsResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 1, result.Page) // Defaults to 1
				suite.AssertExpectations(t)
			},
		},
		{
			name: "default per_page when zero",
			query: queries.ListPendingReportsQuery{
				Page:    1,
				PerPage: 0,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ReportRepo.On("FindPending", mock.Anything, mock.Anything).Return([]*moderation.Report{}, int64(0), nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListPendingReportsResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 20, result.PerPage) // Default per page is 20
				suite.AssertExpectations(t)
			},
		},
		{
			name: "max per_page enforced",
			query: queries.ListPendingReportsQuery{
				Page:    1,
				PerPage: 200, // Over max of 100
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ReportRepo.On("FindPending", mock.Anything, mock.Anything).Return([]*moderation.Report{}, int64(0), nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListPendingReportsResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 100, result.PerPage) // Capped at 100
				suite.AssertExpectations(t)
			},
		},
		{
			name: "repository error",
			query: queries.ListPendingReportsQuery{
				Page:    1,
				PerPage: 10,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ReportRepo.On("FindPending", mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("database error")).Once()
			},
			wantErr: "find pending reports",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListPendingReportsResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find pending reports")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "reports have correct DTO fields",
			query: queries.ListPendingReportsQuery{
				Page:    1,
				PerPage: 10,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				report := testhelpers.ValidReport(t)
				reports := []*moderation.Report{report}

				suite.ReportRepo.On("FindPending", mock.Anything, mock.Anything).Return(reports, int64(1), nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ListPendingReportsResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.Reports, 1)

				dto := result.Reports[0]
				assert.NotEmpty(t, dto.ID)
				assert.Equal(t, testhelpers.ValidUserID, dto.ReporterID)
				assert.Equal(t, testhelpers.ValidImageID, dto.ImageID)
				assert.Equal(t, testhelpers.ValidReportReason, dto.Reason)
				assert.Equal(t, testhelpers.ValidReportDesc, dto.Description)
				assert.Equal(t, "pending", dto.Status)
				assert.False(t, dto.CreatedAt.IsZero())
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := queries.NewListPendingReportsHandler(suite.ReportRepo, &suite.Logger)
			result, err := handler.Handle(context.Background(), tt.query)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
