package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/nsfw"
)

func TestScanImageNSFWHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.ScanImageNSFWCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ScanImageNSFWResult, err error)
	}{
		{
			name: "successful scan - safe image",
			cmd: commands.ScanImageNSFWCommand{
				ImageID:  testhelpers.ValidImageID,
				ImageURL: "http://example.com/safe.jpg",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ImageRepo.On("FindByID", mock.Anything, testhelpers.ValidImageIDParsed()).Return(testhelpers.ValidImage(t), nil).Once()
				suite.NSFWScanRepo.On("HasActiveScan", mock.Anything, testhelpers.ValidImageIDParsed()).Return(false, nil).Once()
				suite.NSFWScanRepo.On("NextID").Return(moderation.NSFWScanID{}).Once()
				suite.NSFWClient.On("Provider").Return(moderation.ProviderSightEngine).Once()
				suite.NSFWScanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Twice() // Initial + Complete
				suite.NSFWClient.On("Scan", mock.Anything, "http://example.com/safe.jpg").Return(&nsfw.ScanResult{
					Category:     moderation.CategorySafe,
					Score:        0.05,
					ScanDuration: 100 * time.Millisecond,
					Provider:     moderation.ProviderSightEngine,
				}, nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ScanImageNSFWResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "safe", result.Category)
				assert.Equal(t, 0.05, result.Score)
				assert.False(t, result.IsNSFW)
				assert.False(t, result.RequiresReview)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful scan - nsfw image",
			cmd: commands.ScanImageNSFWCommand{
				ImageID:  testhelpers.ValidImageID,
				ImageURL: "http://example.com/nsfw.jpg",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ImageRepo.On("FindByID", mock.Anything, testhelpers.ValidImageIDParsed()).Return(testhelpers.ValidImage(t), nil).Once()
				suite.NSFWScanRepo.On("HasActiveScan", mock.Anything, testhelpers.ValidImageIDParsed()).Return(false, nil).Once()
				suite.NSFWScanRepo.On("NextID").Return(moderation.NSFWScanID{}).Once()
				suite.NSFWClient.On("Provider").Return(moderation.ProviderSightEngine).Once()
				suite.NSFWScanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Twice()
				suite.NSFWClient.On("Scan", mock.Anything, "http://example.com/nsfw.jpg").Return(&nsfw.ScanResult{
					Category:     moderation.CategoryNudity,
					Score:        0.95,
					ScanDuration: 100 * time.Millisecond,
					Provider:     moderation.ProviderSightEngine,
				}, nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ScanImageNSFWResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "nudity", result.Category)
				assert.Equal(t, 0.95, result.Score)
				assert.True(t, result.IsNSFW)
				assert.True(t, result.RequiresReview)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "image not found",
			cmd: commands.ScanImageNSFWCommand{
				ImageID:  testhelpers.ValidImageID,
				ImageURL: "http://example.com/image.jpg",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ImageRepo.On("FindByID", mock.Anything, testhelpers.ValidImageIDParsed()).Return(nil, errors.New("not found")).Once()
			},
			wantErr: "find image",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ScanImageNSFWResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find image")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "active scan already exists",
			cmd: commands.ScanImageNSFWCommand{
				ImageID:  testhelpers.ValidImageID,
				ImageURL: "http://example.com/image.jpg",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ImageRepo.On("FindByID", mock.Anything, testhelpers.ValidImageIDParsed()).Return(testhelpers.ValidImage(t), nil).Once()
				suite.NSFWScanRepo.On("HasActiveScan", mock.Anything, testhelpers.ValidImageIDParsed()).Return(true, nil).Once()
			},
			wantErr: "active scan in progress",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ScanImageNSFWResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "active scan in progress")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "force scan even if active exists",
			cmd: commands.ScanImageNSFWCommand{
				ImageID:  testhelpers.ValidImageID,
				ImageURL: "http://example.com/safe.jpg",
				Force:    true,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ImageRepo.On("FindByID", mock.Anything, testhelpers.ValidImageIDParsed()).Return(testhelpers.ValidImage(t), nil).Once()
				// HasActiveScan should NOT be called
				suite.NSFWScanRepo.On("NextID").Return(moderation.NSFWScanID{}).Once()
				suite.NSFWClient.On("Provider").Return(moderation.ProviderSightEngine).Once()
				suite.NSFWScanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Twice()
				suite.NSFWClient.On("Scan", mock.Anything, "http://example.com/safe.jpg").Return(&nsfw.ScanResult{
					Category:     moderation.CategorySafe,
					Score:        0.01,
					ScanDuration: 100 * time.Millisecond,
					Provider:     moderation.ProviderSightEngine,
				}, nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ScanImageNSFWResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "provider scan error",
			cmd: commands.ScanImageNSFWCommand{
				ImageID:  testhelpers.ValidImageID,
				ImageURL: "http://example.com/error.jpg",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.ImageRepo.On("FindByID", mock.Anything, testhelpers.ValidImageIDParsed()).Return(testhelpers.ValidImage(t), nil).Once()
				suite.NSFWScanRepo.On("HasActiveScan", mock.Anything, testhelpers.ValidImageIDParsed()).Return(false, nil).Once()
				suite.NSFWScanRepo.On("NextID").Return(moderation.NSFWScanID{}).Once()
				suite.NSFWClient.On("Provider").Return(moderation.ProviderSightEngine).Once()
				suite.NSFWScanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Twice() // Initial + Failed state
				suite.NSFWClient.On("Scan", mock.Anything, "http://example.com/error.jpg").Return(nil, errors.New("api error")).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "NSFW scan failed",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.ScanImageNSFWResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "NSFW scan failed")
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := commands.NewScanImageNSFWHandler(
				suite.NSFWScanRepo,
				suite.ImageRepo,
				suite.NSFWClient,
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
