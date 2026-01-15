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
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestCreateReportHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.CreateReportCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error)
	}{
		{
			name: "successful report creation",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     testhelpers.ValidImageID,
				Reason:      testhelpers.ValidReportReason,
				Description: testhelpers.ValidReportDesc,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				image := testhelpers.ValidImage(t) // Owned by ValidAdminID, not reporter

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotEmpty(t, result.ReportID)
				assert.Equal(t, "pending", result.Status)
				assert.False(t, result.CreatedAt.IsZero())
				suite.AssertExpectations(t)
			},
		},
		{
			name: "report with inappropriate reason",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     testhelpers.ValidImageID,
				Reason:      "inappropriate",
				Description: "Contains inappropriate content",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				image := testhelpers.ValidImage(t)

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid reporter id",
			cmd: commands.CreateReportCommand{
				ReporterID:  "invalid-uuid",
				ImageID:     testhelpers.ValidImageID,
				Reason:      testhelpers.ValidReportReason,
				Description: testhelpers.ValidReportDesc,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid reporter id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid reporter id")
			},
		},
		{
			name: "invalid image id",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     "invalid-uuid",
				Reason:      testhelpers.ValidReportReason,
				Description: testhelpers.ValidReportDesc,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid image id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid image id")
			},
		},
		{
			name: "image not found",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     testhelpers.ValidImageID,
				Reason:      testhelpers.ValidReportReason,
				Description: testhelpers.ValidReportDesc,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(nil, gallery.ErrImageNotFound).Once()
			},
			wantErr: "find image",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find image")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "cannot report own content",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     testhelpers.ValidImageID,
				Reason:      testhelpers.ValidReportReason,
				Description: testhelpers.ValidReportDesc,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				image := testhelpers.ValidImageOwnedByReporter(t) // Owned by reporter

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: "cannot report your own content",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "cannot report your own content")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid report reason",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     testhelpers.ValidImageID,
				Reason:      "invalid_reason_that_does_not_exist",
				Description: testhelpers.ValidReportDesc,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				image := testhelpers.ValidImage(t)

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: "invalid report reason",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid report reason")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "repository save error",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     testhelpers.ValidImageID,
				Reason:      testhelpers.ValidReportReason,
				Description: testhelpers.ValidReportDesc,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				image := testhelpers.ValidImage(t)

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("database error")).Once()
			},
			wantErr: "save report",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "save report")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "event publish error does not fail operation",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     testhelpers.ValidImageID,
				Reason:      testhelpers.ValidReportReason,
				Description: testhelpers.ValidReportDesc,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				image := testhelpers.ValidImage(t)

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(errors.New("event bus error")).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.NoError(t, err) // Operation succeeds even if event publishing fails
				require.NotNil(t, result)
				assert.NotEmpty(t, result.ReportID)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "empty reporter id",
			cmd: commands.CreateReportCommand{
				ReporterID:  "",
				ImageID:     testhelpers.ValidImageID,
				Reason:      testhelpers.ValidReportReason,
				Description: testhelpers.ValidReportDesc,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid reporter id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name: "empty image id",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     "",
				Reason:      testhelpers.ValidReportReason,
				Description: testhelpers.ValidReportDesc,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid image id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name: "empty description is allowed",
			cmd: commands.CreateReportCommand{
				ReporterID:  testhelpers.ValidUserID,
				ImageID:     testhelpers.ValidImageID,
				Reason:      testhelpers.ValidReportReason,
				Description: "", // Empty description
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				image := testhelpers.ValidImage(t)

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.CreateReportResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := commands.NewCreateReportHandler(
				suite.ReportRepo,
				suite.ImageRepo,
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
