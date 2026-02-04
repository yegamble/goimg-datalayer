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

func TestGetNSFWScanHandler_Handle(t *testing.T) {
	t.Parallel()

	validScanID := "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name    string
		query   queries.GetNSFWScanQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.NSFWScanDTO, err error)
	}{
		{
			name: "successful retrieval",
			query: queries.GetNSFWScanQuery{
				ScanID: validScanID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				scanID, _ := moderation.ParseNSFWScanID(validScanID)
				imageID := testhelpers.ValidImageIDParsed()
				scan := moderation.NewNSFWScan(scanID, imageID, moderation.ProviderSightEngine)

				// Manually set private fields or state if possible, or use constructor.
				// Since we can't set private fields easily without helpers, we rely on public methods
				_ = scan.MarkScanning()
				_ = scan.Complete(moderation.CategorySafe, 0.01, moderation.NSFWDetails{})

				suite.NSFWScanRepo.On("FindByID", mock.Anything, scanID).Return(scan, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.NSFWScanDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, validScanID, result.ID)
				assert.Equal(t, "completed", result.Status)
				assert.Equal(t, "safe", result.Category)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "scan not found",
			query: queries.GetNSFWScanQuery{
				ScanID: validScanID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				scanID, _ := moderation.ParseNSFWScanID(validScanID)
				suite.NSFWScanRepo.On("FindByID", mock.Anything, scanID).Return(nil, errors.New("not found")).Once()
			},
			wantErr: "find scan by id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.NSFWScanDTO, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find scan by id")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid scan id",
			query: queries.GetNSFWScanQuery{
				ScanID: "invalid-uuid",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid scan id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.NSFWScanDTO, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid scan id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := queries.NewGetNSFWScanHandler(suite.NSFWScanRepo)
			result, err := handler.Handle(context.Background(), tt.query)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
