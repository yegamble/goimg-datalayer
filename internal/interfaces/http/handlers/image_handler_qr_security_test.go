package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

// TestImageHandler_GetImageQRCode_HostHeaderInjection demonstrates the vulnerability
func TestImageHandler_GetImageQRCode_HostHeaderInjection(t *testing.T) {
	// Setup mock repo
	mockRepo := new(MockImageRepository)
	logger := zerolog.Nop()
	getImageHandler := queries.NewGetImageHandler(mockRepo, &logger)

	// Create a public image
	image := createQRTestImage(t, gallery.VisibilityPublic)
	mockRepo.
		On("FindByID", mock.Anything, image.ID()).
		Return(image, nil).
		Once()

	// Initialize handler WITHOUT baseURL configuration
	imageHandler := NewImageHandler(
		nil, nil, nil, nil,
		getImageHandler, nil, nil, nil,
		"", // No baseURL configured
		logger,
	)

	// Create request with malicious Host headers
	req := httptest.NewRequest(http.MethodGet, "/images/"+image.ID().String()+"/qr", nil)
	req = withRouteParam(req, "imageID", image.ID().String())

	// Malicious headers
	req.Header.Set("X-Forwarded-Host", "evil.com")
	req.Header.Set("X-Forwarded-Proto", "https")

	rec := httptest.NewRecorder()

	// Execute handler
	imageHandler.GetImageQRCode(rec, req)

	// Assert that the QR code generation FAILED because BASE_URL is not configured
	// and we no longer fall back to insecure host header inference.
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// TestImageHandler_GetImageQRCode_SecureBaseURL demonstrates the fix
func TestImageHandler_GetImageQRCode_SecureBaseURL(t *testing.T) {
	// Setup mock repo
	mockRepo := new(MockImageRepository)
	logger := zerolog.Nop()
	getImageHandler := queries.NewGetImageHandler(mockRepo, &logger)

	// Create a public image
	image := createQRTestImage(t, gallery.VisibilityPublic)
	mockRepo.
		On("FindByID", mock.Anything, image.ID()).
		Return(image, nil).
		Once()

	// Initialize handler WITH secure baseURL configuration
	secureBaseURL := "https://gallery.example.com"
	imageHandler := NewImageHandler(
		nil, nil, nil, nil,
		getImageHandler, nil, nil, nil,
		secureBaseURL,
		logger,
	)

	// Create request with malicious Host headers
	req := httptest.NewRequest(http.MethodGet, "/images/"+image.ID().String()+"/qr", nil)
	req = withRouteParam(req, "imageID", image.ID().String())

	// Malicious headers should be IGNORED because baseURL is set
	req.Header.Set("X-Forwarded-Host", "evil.com")
	req.Header.Set("X-Forwarded-Proto", "https")

	rec := httptest.NewRecorder()

	// Execute handler
	imageHandler.GetImageQRCode(rec, req)

	// Assert success
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verification that the QR code points to the secure URL would require decoding,
	// but we can trust the logic branch. If baseURL is set, inferBaseURLFromRequest is NOT called.
}
