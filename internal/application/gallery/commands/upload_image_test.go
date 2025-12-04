package commands_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestUploadImageHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.UploadImageCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error)
	}{
		{
			name: "successful upload",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    testhelpers.ValidFilename,
				Title:       testhelpers.ValidTitle,
				Description: testhelpers.ValidDescription,
				Visibility:  "public",
				Tags:        []string{"nature", "landscape"},
				MimeType:    testhelpers.ValidMimeType,
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Mock NextID
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("NextID").Return(imageID).Once()

				// Mock storage provider
				suite.Storage.On("Provider").Return("local").Maybe()

				// Mock storage Put
				suite.Storage.On("Put", mock.Anything, mock.MatchedBy(func(key string) bool {
					return key != ""
				}), mock.Anything, testhelpers.ValidFileSize, mock.Anything).Return(nil).Once()

				// Mock Save
				suite.ImageRepo.On("Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
					return img.ID().String() == imageID.String() &&
						img.Status() == gallery.StatusProcessing &&
						len(img.Tags()) == 2
				})).Return(nil).Once()

				// Mock event publishing
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()

				// Mock job enqueueing
				suite.JobEnqueuer.On("EnqueueImageProcessing", mock.Anything, imageID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidImageID, result.ImageID)
				assert.Equal(t, gallery.StatusProcessing.String(), result.Status)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid user id",
			cmd: commands.UploadImageCommand{
				UserID:      "invalid-uuid",
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks needed - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid user id")
				assert.Nil(t, result)
			},
		},
		{
			name: "file size too large",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    gallery.MaxFileSize + 1, // Exceeds limit
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks needed - should fail validation
			},
			wantErr: gallery.ErrFileTooLarge,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrFileTooLarge)
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid file size - zero",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    0, // Invalid
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks needed - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid file size")
				assert.Nil(t, result)
			},
		},
		{
			name: "unsupported MIME type",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    "test.pdf",
				MimeType:    "application/pdf", // Not supported
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks needed - should fail validation
			},
			wantErr: gallery.ErrInvalidMimeType,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrInvalidMimeType)
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid visibility",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Visibility:  "invalid", // Invalid visibility
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks needed - should fail validation
			},
			wantErr: gallery.ErrInvalidVisibility,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrInvalidVisibility)
				assert.Nil(t, result)
			},
		},
		{
			name: "storage upload failure",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("NextID").Return(imageID).Once()
				suite.Storage.On("Provider").Return("local").Maybe()

				// Storage Put fails
				suite.Storage.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("storage error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "store image")
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid tag",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Tags:        []string{"a"}, // Too short (min 2 chars)
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("NextID").Return(imageID).Once()
				suite.Storage.On("Provider").Return("local").Maybe()
				suite.Storage.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Once()
			},
			wantErr: gallery.ErrTagTooShort,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrTagTooShort)
				assert.Nil(t, result)
			},
		},
		{
			name: "too many tags",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Tags: []string{
					"tag1", "tag2", "tag3", "tag4", "tag5",
					"tag6", "tag7", "tag8", "tag9", "tag10",
					"tag11", "tag12", "tag13", "tag14", "tag15",
					"tag16", "tag17", "tag18", "tag19", "tag20",
					"tag21", // 21 tags, max is 20
				},
				Width:  testhelpers.ValidWidth,
				Height: testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("NextID").Return(imageID).Once()
				suite.Storage.On("Provider").Return("local").Maybe()
				suite.Storage.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Once()
			},
			wantErr: gallery.ErrTooManyTags,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrTooManyTags)
				assert.Nil(t, result)
			},
		},
		{
			name: "repository save failure",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("NextID").Return(imageID).Once()
				suite.Storage.On("Provider").Return("local").Maybe()
				suite.Storage.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Once()

				// Repository Save fails
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "save image")
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "event publishing failure - should still succeed",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("NextID").Return(imageID).Once()
				suite.Storage.On("Provider").Return("local").Maybe()
				suite.Storage.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()

				// Event publishing fails (non-critical)
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).
					Return(fmt.Errorf("event bus unavailable")).Maybe()

				suite.JobEnqueuer.On("EnqueueImageProcessing", mock.Anything, imageID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				// Should still succeed even if event publishing fails
				require.NoError(t, err)
				require.NotNil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "job enqueueing failure - should still succeed",
			cmd: commands.UploadImageCommand{
				UserID:      testhelpers.ValidUserID,
				FileContent: testhelpers.ValidFileReader(),
				FileSize:    testhelpers.ValidFileSize,
				Filename:    testhelpers.ValidFilename,
				MimeType:    testhelpers.ValidMimeType,
				Width:       testhelpers.ValidWidth,
				Height:      testhelpers.ValidHeight,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("NextID").Return(imageID).Once()
				suite.Storage.On("Provider").Return("local").Maybe()
				suite.Storage.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()

				// Job enqueueing fails (non-critical)
				suite.JobEnqueuer.On("EnqueueImageProcessing", mock.Anything, imageID.String()).
					Return(fmt.Errorf("queue unavailable")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UploadImageResult, err error) {
				// Should still succeed even if job enqueueing fails
				require.NoError(t, err)
				require.NotNil(t, result)
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

			handler := commands.NewUploadImageHandler(
				suite.ImageRepo,
				suite.Storage,
				suite.JobEnqueuer,
				suite.EventPublisher,
				&suite.Logger,
			)

			// Act
			result, err := handler.Handle(context.Background(), tt.cmd)

			// Assert
			if tt.assert != nil {
				tt.assert(t, suite, result, err)
			} else if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

// TestUploadImageCommand_Interface verifies the command implements the interface.

// BenchmarkUploadImageHandler_Handle benchmarks the upload handler.
func BenchmarkUploadImageHandler_Handle(b *testing.B) {
	suite := &testhelpers.TestSuite{
		ImageRepo:      new(testhelpers.MockImageRepository),
		Storage:        new(testhelpers.MockStorage),
		JobEnqueuer:    new(testhelpers.MockJobEnqueuer),
		EventPublisher: new(testhelpers.MockEventPublisher),
		Logger:         zerolog.Nop(),
	}

	imageID := testhelpers.ValidImageIDParsed()
	suite.ImageRepo.On("NextID").Return(imageID)
	suite.Storage.On("Provider").Return("local").Maybe()
	suite.Storage.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
	suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
	suite.JobEnqueuer.On("EnqueueImageProcessing", mock.Anything, imageID.String()).Return(nil)

	handler := commands.NewUploadImageHandler(
		suite.ImageRepo,
		suite.Storage,
		suite.JobEnqueuer,
		suite.EventPublisher,
		&suite.Logger,
	)

	cmd := commands.UploadImageCommand{
		UserID:      testhelpers.ValidUserID,
		FileContent: struct{ io.Reader }{testhelpers.ValidFileReader()},
		FileSize:    testhelpers.ValidFileSize,
		Filename:    testhelpers.ValidFilename,
		MimeType:    testhelpers.ValidMimeType,
		Width:       testhelpers.ValidWidth,
		Height:      testhelpers.ValidHeight,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.Handle(context.Background(), cmd)
	}
}
