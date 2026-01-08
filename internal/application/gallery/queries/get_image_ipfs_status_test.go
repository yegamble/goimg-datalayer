package queries_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

const testGatewayURL = "https://ipfs.io/ipfs/" + testhelpers.ValidIPFSCID

func TestGetImageIPFSStatusHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   queries.GetImageIPFSStatusQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error)
	}{
		{
			name: "successful status check - owner with IPFS",
			query: queries.GetImageIPFSStatusQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImageWithIPFS(t)
				imageID := testhelpers.ValidImageIDParsed()
				cid := image.IPFSMetadata().CID()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.IPFSService.On("GatewayURL", cid).Return(testGatewayURL).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidImageID, result.ImageID)
				assert.True(t, result.IsPinned)
				assert.NotNil(t, result.CID)
				assert.Equal(t, testhelpers.ValidIPFSCID, *result.CID)
				assert.NotNil(t, result.IPFSURI)
				assert.NotNil(t, result.GatewayURL)
				assert.Equal(t, testGatewayURL, *result.GatewayURL)
				assert.NotNil(t, result.PinnedAt)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful status check - owner without IPFS",
			query: queries.GetImageIPFSStatusQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t) // No IPFS
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidImageID, result.ImageID)
				assert.False(t, result.IsPinned)
				assert.Nil(t, result.CID)
				assert.Nil(t, result.IPFSURI)
				assert.Nil(t, result.GatewayURL)
				assert.Nil(t, result.PinnedAt)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful status check - public image anonymous user",
			query: queries.GetImageIPFSStatusQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: "", // No user (anonymous)
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImageWithIPFS(t) // Public image with IPFS
				imageID := testhelpers.ValidImageIDParsed()
				cid := image.IPFSMetadata().CID()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.IPFSService.On("GatewayURL", cid).Return(testGatewayURL).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.True(t, result.IsPinned)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid image id",
			query: queries.GetImageIPFSStatusQuery{
				ImageID:          "invalid-uuid",
				RequestingUserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, _ *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, _ *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid image id")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid requesting user id",
			query: queries.GetImageIPFSStatusQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: "invalid-uuid",
			},
			setup: func(t *testing.T, _ *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, _ *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid requesting user id")
				assert.Nil(t, result)
			},
		},
		{
			name: "image not found",
			query: queries.GetImageIPFSStatusQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("FindByID", mock.Anything, imageID).
					Return(nil, gallery.ErrImageNotFound).Once()
			},
			wantErr: gallery.ErrImageNotFound,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error) {
				require.Error(t, err)
				require.ErrorIs(t, err, gallery.ErrImageNotFound)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "unauthorized - private image non-owner",
			query: queries.GetImageIPFSStatusQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: "550e8400-e29b-41d4-a716-446655440001", // Different user
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImageWithIPFS(t)
				imageID := testhelpers.ValidImageIDParsed()

				// Make it private
				require.NoError(t, image.UpdateVisibility(gallery.VisibilityPrivate))

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrUnauthorizedAccess,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error) {
				require.Error(t, err)
				require.ErrorIs(t, err, gallery.ErrUnauthorizedAccess)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "unauthorized - private image anonymous user",
			query: queries.GetImageIPFSStatusQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: "", // No user (anonymous)
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImageWithIPFS(t)
				imageID := testhelpers.ValidImageIDParsed()

				// Make it private
				require.NoError(t, image.UpdateVisibility(gallery.VisibilityPrivate))

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrUnauthorizedAccess,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error) {
				require.Error(t, err)
				require.ErrorIs(t, err, gallery.ErrUnauthorizedAccess)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "repository failure",
			query: queries.GetImageIPFSStatusQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("FindByID", mock.Anything, imageID).
					Return(nil, fmt.Errorf("database error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageIPFSStatusDTO, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "find image")
				assert.Nil(t, result)
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

			handler := queries.NewGetImageIPFSStatusHandler(
				suite.ImageRepo,
				suite.IPFSService,
				&suite.Logger,
			)

			// Act
			result, err := handler.Handle(context.Background(), tt.query)

			// Assert
			switch {
			case tt.assert != nil:
				tt.assert(t, suite, result, err)
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			default:
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}
