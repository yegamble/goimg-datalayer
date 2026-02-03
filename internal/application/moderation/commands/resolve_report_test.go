package commands_test

import (
	"context"
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
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestResolveReportHandler_Handle(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockReports := new(testhelpers.MockReportRepository)
		mockPublisher := new(testhelpers.MockEventPublisher)
		logger := zerolog.Nop()

		handler := commands.NewResolveReportHandler(
			mockReports,
			mockPublisher,
			&logger,
		)

		reportID := moderation.NewReportID()
		resolverID := identity.NewUserID()

		// Create a dummy report in Reviewing status
		report := moderation.ReconstructReport(
			reportID,
			identity.NewUserID(), // reporter
			gallery.NewImageID(), // image
			moderation.ReasonSpam,
			"Spam content",
			moderation.StatusReviewing,
			nil, nil, "",
			time.Now(),
		)

		cmd := commands.ResolveReportCommand{
			ReportID:   reportID.String(),
			ResolverID: resolverID.String(),
			Resolution: "Banned user",
		}

		mockReports.On("FindByID", mock.Anything, reportID).Return(report, nil)
		mockReports.On("Save", mock.Anything, mock.Anything).Return(nil)
		mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.Equal(t, reportID.String(), result.ReportID)
		assert.Equal(t, "resolved", result.Status)
		assert.Equal(t, "Banned user", result.Resolution)

		mockReports.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})
}
