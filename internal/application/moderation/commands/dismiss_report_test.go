package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestDismissReportHandler_Handle(t *testing.T) {
	t.Run("successfully dismisses a pending report", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewDismissReportHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		report := testhelpers.ValidReport(t)
		reportID := report.ID()
		resolverID := testhelpers.ValidAdminIDParsed()

		suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil)
		suite.ReportRepo.On("Save", mock.Anything, mock.MatchedBy(func(r *moderation.Report) bool {
			return r.Status() == moderation.StatusDismissed
		})).Return(nil)
		suite.EventPublisher.On("Publish", mock.Anything, mock.MatchedBy(func(e moderation.ReportDismissed) bool {
			return e.ReportID == reportID && e.ResolverID == resolverID
		})).Return(nil)

		cmd := commands.DismissReportCommand{
			ReportID:   reportID.String(),
			ResolverID: resolverID.String(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reportID.String(), result.ReportID)
		assert.Equal(t, moderation.StatusDismissed.String(), result.Status)

		suite.AssertExpectations(t)
	})

	t.Run("successfully dismisses a reviewing report", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewDismissReportHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		report := testhelpers.ValidReportInReviewing(t)
		reportID := report.ID()
		resolverID := testhelpers.ValidAdminIDParsed()

		suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil)
		suite.ReportRepo.On("Save", mock.Anything, mock.MatchedBy(func(r *moderation.Report) bool {
			return r.Status() == moderation.StatusDismissed
		})).Return(nil)
		suite.EventPublisher.On("Publish", mock.Anything, mock.MatchedBy(func(e moderation.ReportDismissed) bool {
			return e.ReportID == reportID && e.ResolverID == resolverID
		})).Return(nil)

		cmd := commands.DismissReportCommand{
			ReportID:   reportID.String(),
			ResolverID: resolverID.String(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reportID.String(), result.ReportID)
		assert.Equal(t, moderation.StatusDismissed.String(), result.Status)

		suite.AssertExpectations(t)
	})

	t.Run("fails when report not found", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewDismissReportHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		reportID := testhelpers.ValidReportIDParsed()
		resolverID := testhelpers.ValidAdminIDParsed()

		suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(nil, errors.New("not found"))

		cmd := commands.DismissReportCommand{
			ReportID:   reportID.String(),
			ResolverID: resolverID.String(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "find report")

		suite.AssertExpectations(t)
	})

	t.Run("fails when invalid report ID", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewDismissReportHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		cmd := commands.DismissReportCommand{
			ReportID:   "invalid-uuid",
			ResolverID: testhelpers.ValidAdminID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid report id")
	})

	t.Run("fails when invalid resolver ID", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewDismissReportHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		cmd := commands.DismissReportCommand{
			ReportID:   testhelpers.ValidReportID,
			ResolverID: "invalid-uuid",
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid resolver id")
	})

	t.Run("fails when report already resolved", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewDismissReportHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		report := testhelpers.ValidResolvedReport(t)
		reportID := report.ID()
		resolverID := testhelpers.ValidAdminIDParsed()

		suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil)

		cmd := commands.DismissReportCommand{
			ReportID:   reportID.String(),
			ResolverID: resolverID.String(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "dismiss report")
		assert.True(t, errors.Is(err, moderation.ErrReportInTerminalState) || assert.Contains(t, err.Error(), "already in terminal state"))

		suite.AssertExpectations(t)
	})

	t.Run("fails when report already dismissed", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewDismissReportHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		report := testhelpers.ValidDismissedReport(t)
		reportID := report.ID()
		resolverID := testhelpers.ValidAdminIDParsed()

		suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil)

		cmd := commands.DismissReportCommand{
			ReportID:   reportID.String(),
			ResolverID: resolverID.String(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "dismiss report")

		suite.AssertExpectations(t)
	})
}
