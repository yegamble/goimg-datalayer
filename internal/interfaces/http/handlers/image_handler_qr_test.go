package handlers

import (
	"bytes"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestImageHandler_GetImageQRCode_DefaultSize(t *testing.T) {
	mockRepo := new(MockImageRepository)
	logger := zerolog.Nop()
	getImageHandler := queries.NewGetImageHandler(mockRepo, &logger)

	image := createQRTestImage(t, gallery.VisibilityPublic)
	mockRepo.
		On("FindByID", mock.Anything, image.ID()).
		Return(image, nil).
		Once()

	imageHandler := NewImageHandler(
		nil, nil, nil, nil,
		getImageHandler, nil, nil, nil,
		"https://example.com",
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/images/"+image.ID().String()+"/qr", nil)
	req.Host = "api.example.com"
	req = withRouteParam(req, "imageID", image.ID().String())
	rec := httptest.NewRecorder()

	imageHandler.GetImageQRCode(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	require.Equal(t, "public, max-age=3600", rec.Header().Get("Cache-Control"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), image.ID().String())
	assert.NotEmpty(t, rec.Body.Bytes())

	decoded, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
	require.NoError(t, err)
	assert.Equal(t, defaultQRCodeSize, decoded.Bounds().Dx())
	assert.Equal(t, defaultQRCodeSize, decoded.Bounds().Dy())
	mockRepo.AssertExpectations(t)
}

func TestImageHandler_GetImageQRCode_SizeOutOfRange(t *testing.T) {
	logger := zerolog.Nop()
	imageHandler := NewImageHandler(
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		"",
		logger,
	)

	testCases := []struct {
		name string
		size string
	}{
		{name: "below minimum", size: "127"},
		{name: "above maximum", size: "1025"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/images/00000000-0000-0000-0000-000000000000/qr?size="+tc.size, nil)
			req = withRouteParam(req, "imageID", "00000000-0000-0000-0000-000000000000")
			rec := httptest.NewRecorder()

			imageHandler.GetImageQRCode(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestImageHandler_GetImageQRCode_InvalidImageID(t *testing.T) {
	logger := zerolog.Nop()
	imageHandler := NewImageHandler(
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		"",
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/images/not-a-uuid/qr", nil)
	req = withRouteParam(req, "imageID", "not-a-uuid")
	rec := httptest.NewRecorder()

	imageHandler.GetImageQRCode(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestImageHandler_GetImageQRCode_ImageNotFound(t *testing.T) {
	mockRepo := new(MockImageRepository)
	logger := zerolog.Nop()
	getImageHandler := queries.NewGetImageHandler(mockRepo, &logger)

	const imageID = "00000000-0000-0000-0000-000000000000"
	parsedID, err := gallery.ParseImageID(imageID)
	require.NoError(t, err)

	mockRepo.
		On("FindByID", mock.Anything, parsedID).
		Return(nil, gallery.ErrImageNotFound).
		Once()

	imageHandler := NewImageHandler(
		nil, nil, nil, nil,
		getImageHandler, nil, nil, nil,
		"",
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/images/"+imageID+"/qr", nil)
	req = withRouteParam(req, "imageID", imageID)
	rec := httptest.NewRecorder()

	imageHandler.GetImageQRCode(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	mockRepo.AssertExpectations(t)
}

func TestGenerateQRCodePNG(t *testing.T) {
	pngData, err := generateQRCodePNG("https://example.com/images/abc/preview", 300)
	require.NoError(t, err)
	require.NotEmpty(t, pngData)

	decoded, err := png.Decode(bytes.NewReader(pngData))
	require.NoError(t, err)
	assert.Equal(t, 300, decoded.Bounds().Dx())
	assert.Equal(t, 300, decoded.Bounds().Dy())
}
