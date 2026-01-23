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

func TestStartReviewHandler_Handle(t *testing.T) {
	t.Run("successfully starts review on pending report", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewStartReviewHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		report := testhelpers.ValidReport(t)
		reportID := report.ID()

		suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil)
		suite.ReportRepo.On("Save", mock.Anything, mock.MatchedBy(func(r *moderation.Report) bool {
			return r.Status() == moderation.StatusReviewing
		})).Return(nil)
		suite.EventPublisher.On("Publish", mock.Anything, mock.MatchedBy(func(e moderation.ReportReviewStarted) bool {
			return e.ReportID == reportID
		})).Return(nil)

		cmd := commands.StartReviewCommand{
			ReportID: reportID.String(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reportID.String(), result.ReportID)
		assert.Equal(t, moderation.StatusReviewing.String(), result.Status)

		suite.AssertExpectations(t)
	})

	t.Run("fails when report not found", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewStartReviewHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		reportID := testhelpers.ValidReportIDParsed()

		suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(nil, errors.New("not found"))

		cmd := commands.StartReviewCommand{
			ReportID: reportID.String(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "find report")

		suite.AssertExpectations(t)
	})

	t.Run("fails when invalid report ID", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewStartReviewHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		cmd := commands.StartReviewCommand{
			ReportID: "invalid-uuid",
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid report id")
	})

	t.Run("succeeds idempotently when report already in review", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewStartReviewHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		report := testhelpers.ValidReportInReviewing(t)
		reportID := report.ID()

		suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil)
		suite.ReportRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

		cmd := commands.StartReviewCommand{
			ReportID: reportID.String(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reportID.String(), result.ReportID)
		assert.Equal(t, moderation.StatusReviewing.String(), result.Status)

		suite.AssertExpectations(t)
	})

	t.Run("fails when report already resolved", func(t *testing.T) {
		suite := testhelpers.NewTestSuite(t)
		handler := commands.NewStartReviewHandler(suite.ReportRepo, suite.EventPublisher, &suite.Logger)

		report := testhelpers.ValidResolvedReport(t)
		reportID := report.ID()

		suite.ReportRepo.On("FindByID", mock.Anything, reportID).Return(report, nil)

		cmd := commands.StartReviewCommand{
			ReportID: reportID.String(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "start review")
		assert.True(t, errors.Is(err, moderation.ErrReportInTerminalState) || assert.Contains(t, err.Error(), "already in terminal state"))

		suite.AssertExpectations(t)
	})
}
