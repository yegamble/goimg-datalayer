package commands_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestCreateReportHandler_Handle(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockReports := new(testhelpers.MockReportRepository)
		mockImages := new(testhelpers.MockImageRepository)
		mockPublisher := new(testhelpers.MockEventPublisher)
		logger := zerolog.Nop()

		handler := commands.NewCreateReportHandler(
			mockReports,
			mockImages,
			mockPublisher,
			&logger,
		)

		reporterID := identity.NewUserID()
		ownerID := identity.NewUserID() // Different from reporter
		imageID := gallery.NewImageID()

		// Create a dummy image
		image := createDummyImage(t, imageID, ownerID)

		cmd := commands.CreateReportCommand{
			ReporterID:  reporterID.String(),
			ImageID:     imageID.String(),
			Reason:      "spam",
			Description: "This is spam",
		}

		mockImages.On("FindByID", mock.Anything, imageID).Return(image, nil)
		mockReports.On("Save", mock.Anything, mock.Anything).Return(nil)
		mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

		// Act
		result, err := handler.Handle(context.Background(), cmd)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, result.ReportID)
		assert.Equal(t, "pending", result.Status)

		mockImages.AssertExpectations(t)
		mockReports.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("SelfReport", func(t *testing.T) {
		// Arrange
		mockReports := new(testhelpers.MockReportRepository)
		mockImages := new(testhelpers.MockImageRepository)
		mockPublisher := new(testhelpers.MockEventPublisher)
		logger := zerolog.Nop()

		handler := commands.NewCreateReportHandler(
			mockReports,
			mockImages,
			mockPublisher,
			&logger,
		)

		reporterID := identity.NewUserID()
		imageID := gallery.NewImageID()

		// Reporter is owner
		image := createDummyImage(t, imageID, reporterID)

		cmd := commands.CreateReportCommand{
			ReporterID:  reporterID.String(),
			ImageID:     imageID.String(),
			Reason:      "spam",
			Description: "I hate my own image",
		}

		mockImages.On("FindByID", mock.Anything, imageID).Return(image, nil)

		// Act
		result, err := handler.Handle(context.Background(), cmd)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "cannot report your own content")
	})

	t.Run("ImageNotFound", func(t *testing.T) {
		mockReports := new(testhelpers.MockReportRepository)
		mockImages := new(testhelpers.MockImageRepository)
		mockPublisher := new(testhelpers.MockEventPublisher)
		logger := zerolog.Nop()

		handler := commands.NewCreateReportHandler(
			mockReports,
			mockImages,
			mockPublisher,
			&logger,
		)

		reporterID := identity.NewUserID()
		imageID := gallery.NewImageID()

		cmd := commands.CreateReportCommand{
			ReporterID:  reporterID.String(),
			ImageID:     imageID.String(),
			Reason:      "spam",
			Description: "Spam",
		}

		mockImages.On("FindByID", mock.Anything, imageID).Return(nil, fmt.Errorf("image not found"))

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// Helpers

func createDummyImage(t *testing.T, id gallery.ImageID, ownerID identity.UserID) *gallery.Image {
	metadata, err := gallery.NewImageMetadata(
		"Test Image",
		"Test Description",
		"test.jpg",
		"image/jpeg",
		800,
		600,
		102400,
		"test/key",
		"local",
	)
	require.NoError(t, err)

	return gallery.ReconstructImage(
		id,
		ownerID,
		metadata,
		gallery.VisibilityPublic,
		gallery.StatusActive,
		gallery.ScanStatusClean,
		[]gallery.ImageVariant{},
		[]gallery.Tag{},
		nil,
		0, 0, 0,
		time.Now(),
		time.Now(),
	)
}
