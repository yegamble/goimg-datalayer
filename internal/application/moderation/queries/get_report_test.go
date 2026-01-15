package queries_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestGetReportHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   queries.GetReportQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ReportDTO, err error)
	}{
		{
			name: "successful retrieval of pending report",
			query: queries.GetReportQuery{
				ReportID: testhelpers.ValidReportID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidReport(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ReportDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidReportID, result.ID)
				assert.Equal(t, testhelpers.ValidUserID, result.ReporterID)
				assert.Equal(t, testhelpers.ValidImageID, result.ImageID)
				assert.Equal(t, testhelpers.ValidReportReason, result.Reason)
				assert.Equal(t, testhelpers.ValidReportDesc, result.Description)
				assert.Equal(t, "pending", result.Status)
				assert.Nil(t, result.ResolvedBy)
				assert.Nil(t, result.ResolvedAt)
				assert.Empty(t, result.Resolution)
				assert.False(t, result.CreatedAt.IsZero())
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful retrieval of resolved report",
			query: queries.GetReportQuery{
				ReportID: testhelpers.ValidReportID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidResolvedReport(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ReportDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "resolved", result.Status)
				assert.NotNil(t, result.ResolvedBy)
				assert.Equal(t, testhelpers.ValidAdminID, *result.ResolvedBy)
				assert.NotNil(t, result.ResolvedAt)
				assert.NotEmpty(t, result.Resolution)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful retrieval of reviewing report",
			query: queries.GetReportQuery{
				ReportID: testhelpers.ValidReportID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidReportInReviewing(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ReportDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "reviewing", result.Status)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid report id",
			query: queries.GetReportQuery{
				ReportID: "invalid-uuid",
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid report id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ReportDTO, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid report id")
			},
		},
		{
			name: "empty report id",
			query: queries.GetReportQuery{
				ReportID: "",
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid report id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ReportDTO, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid report id")
			},
		},
		{
			name: "report not found",
			query: queries.GetReportQuery{
				ReportID: testhelpers.ValidReportID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(nil, moderation.ErrReportNotFound).Once()
			},
			wantErr: "find report by id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ReportDTO, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find report by id")
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := queries.NewGetReportHandler(suite.ReportRepo)
			result, err := handler.Handle(context.Background(), tt.query)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
