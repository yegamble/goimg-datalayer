package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestResolveReportHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.ResolveReportCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error)
	}{
		{
			name: "successful resolution from pending status",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: testhelpers.ValidAdminID,
				Resolution: "Content reviewed and removed",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidReport(t) // In pending status

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
				suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidReportID, result.ReportID)
				assert.Equal(t, "resolved", result.Status)
				assert.Equal(t, "Content reviewed and removed", result.Resolution)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful resolution from reviewing status",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: testhelpers.ValidAdminID,
				Resolution: "Violation confirmed, user warned",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidReportInReviewing(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
				suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "resolved", result.Status)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid report id",
			cmd: commands.ResolveReportCommand{
				ReportID:   "invalid-uuid",
				ResolverID: testhelpers.ValidAdminID,
				Resolution: "Content removed",
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid report id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid report id")
			},
		},
		{
			name: "invalid resolver id",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: "invalid-uuid",
				Resolution: "Content removed",
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid resolver id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid resolver id")
			},
		},
		{
			name: "report not found",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: testhelpers.ValidAdminID,
				Resolution: "Content removed",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(nil, moderation.ErrReportNotFound).Once()
			},
			wantErr: "find report",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find report")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "report already resolved",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: testhelpers.ValidAdminID,
				Resolution: "Additional action taken",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidResolvedReport(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
			},
			wantErr: "resolve report",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "resolve report")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "report already dismissed",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: testhelpers.ValidAdminID,
				Resolution: "Trying to resolve dismissed report",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidDismissedReport(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
			},
			wantErr: "resolve report",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "resolve report")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "empty resolution",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: testhelpers.ValidAdminID,
				Resolution: "",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidReport(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
			},
			wantErr: "resolve report",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "resolve report")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "repository save error",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: testhelpers.ValidAdminID,
				Resolution: "Content reviewed",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidReport(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
				suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("database error")).Once()
			},
			wantErr: "save report",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "save report")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "event publish error does not fail operation",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: testhelpers.ValidAdminID,
				Resolution: "Content reviewed and removed",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidReport(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
				suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(errors.New("event bus error")).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.NoError(t, err) // Operation succeeds even if event publishing fails
				require.NotNil(t, result)
				assert.Equal(t, "resolved", result.Status)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "resolution too long",
			cmd: commands.ResolveReportCommand{
				ReportID:   testhelpers.ValidReportID,
				ResolverID: testhelpers.ValidAdminID,
				Resolution: string(make([]byte, 1001)), // 1001 characters, max is 1000
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				reportID := testhelpers.ValidReportIDParsed()
				report := testhelpers.ValidReport(t)

				suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil).Once()
			},
			wantErr: "resolve report",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ResolveReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "resolve report")
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := commands.NewResolveReportHandler(
				suite.ReportRepo,
				suite.EventPublisher,
				&suite.Logger,
			)

			result, err := handler.Handle(context.Background(), tt.cmd)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
