package handlers

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
)

// OEmbedHandler handles oEmbed protocol requests for image embedding.
// Implements the oEmbed 1.0 specification (https://oembed.com/).
type OEmbedHandler struct {
	getImage *queries.GetImageHandler
	baseURL  string // Base URL for generating embed URLs (e.g., "https://example.com")
	logger   zerolog.Logger
}

// OEmbedResponse represents an oEmbed response for photo type.
// Follows oEmbed 1.0 spec: https://oembed.com/#section2.3.4.1
type OEmbedResponse struct {
	// Required fields
	Type    string `json:"type" xml:"type"`       // Always "photo" for images
	Version string `json:"version" xml:"version"` // Always "1.0"

	// Optional fields (required for photo type)
	URL    string `json:"url" xml:"url"`       // URL of the image
	Width  int    `json:"width" xml:"width"`   // Width in pixels
	Height int    `json:"height" xml:"height"` // Height in pixels

	// Common optional fields
	Title           string `json:"title,omitempty" xml:"title,omitempty"`
	AuthorName      string `json:"author_name,omitempty" xml:"author_name,omitempty"`
	AuthorURL       string `json:"author_url,omitempty" xml:"author_url,omitempty"`
	ProviderName    string `json:"provider_name,omitempty" xml:"provider_name,omitempty"`
	ProviderURL     string `json:"provider_url,omitempty" xml:"provider_url,omitempty"`
	CacheAge        int    `json:"cache_age,omitempty" xml:"cache_age,omitempty"`
	ThumbnailURL    string `json:"thumbnail_url,omitempty" xml:"thumbnail_url,omitempty"`
	ThumbnailWidth  int    `json:"thumbnail_width,omitempty" xml:"thumbnail_width,omitempty"`
	ThumbnailHeight int    `json:"thumbnail_height,omitempty" xml:"thumbnail_height,omitempty"`
}

// OEmbedXMLResponse wraps OEmbedResponse for XML serialization.
type OEmbedXMLResponse struct {
	XMLName xml.Name `xml:"oembed"`
	OEmbedResponse
}

// NewOEmbedHandler creates a new oEmbed handler.
func NewOEmbedHandler(
	getImage *queries.GetImageHandler,
	baseURL string,
	logger zerolog.Logger,
) *OEmbedHandler {
	return &OEmbedHandler{
		getImage: getImage,
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		logger:   logger,
	}
}

// Routes returns a chi.Router with oEmbed routes.
// Mount this at /api/v1/oembed
func (h *OEmbedHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetOEmbed)
	return r
}

// GetOEmbed handles GET /api/v1/oembed
//
// Query parameters:
//   - url (required): URL of the image to embed
//   - format (optional): "json" or "xml" (default: "json")
//   - maxwidth (optional): Maximum width of embedded content
//   - maxheight (optional): Maximum height of embedded content
//
// Supports both JSON and XML response formats per oEmbed spec.
func (h *OEmbedHandler) GetOEmbed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		h.writeError(w, r, http.StatusBadRequest, "url parameter is required")
		return
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "xml" {
		h.writeError(w, r, http.StatusBadRequest, "format must be 'json' or 'xml'")
		return
	}

	// Parse maxwidth and maxheight
	maxWidth, _ := strconv.Atoi(r.URL.Query().Get("maxwidth"))
	maxHeight, _ := strconv.Atoi(r.URL.Query().Get("maxheight"))

	// Extract image ID from URL
	imageID, err := h.extractImageID(urlParam)
	if err != nil {
		h.logger.Debug().Err(err).Str("url", urlParam).Msg("Failed to parse image URL")
		h.writeError(w, r, http.StatusNotFound, "Invalid image URL")
		return
	}

	// Fetch image details
	query := queries.GetImageQuery{
		ImageID: imageID.String(),
		// No user ID - only public images are embeddable
	}
	result, err := h.getImage.Handle(ctx, query)
	if err != nil {
		h.logger.Debug().Err(err).Str("image_id", imageID.String()).Msg("Image not found")
		h.writeError(w, r, http.StatusNotFound, "Image not found")
		return
	}

	// Only allow embedding of public images
	if result.Visibility != "public" {
		h.writeError(w, r, http.StatusNotFound, "Image not found")
		return
	}

	// Build oEmbed response
	response := h.buildOEmbedResponse(result, maxWidth, maxHeight)

	// Write response in requested format
	if format == "xml" {
		h.writeXMLResponse(w, response)
	} else {
		h.writeJSONResponse(w, response)
	}
}

// uuidPattern matches UUID anywhere in path
var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// extractImageID parses the image ID from various URL formats.
// Supports:
//   - /images/{id}
//   - /api/v1/images/{id}
//   - Full URLs with domain
func (h *OEmbedHandler) extractImageID(rawURL string) (uuid.UUID, error) {
	// Parse URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid URL: %w", err)
	}

	path := parsed.Path

	// Try to extract UUID from path using regex
	match := uuidPattern.FindString(path)
	if match == "" {
		return uuid.Nil, fmt.Errorf("no image ID found in URL path")
	}

	return uuid.Parse(match)
}

// buildOEmbedResponse creates the oEmbed response from image data.
func (h *OEmbedHandler) buildOEmbedResponse(
	image *queries.ImageDTO,
	maxWidth, maxHeight int,
) OEmbedResponse {
	// Calculate dimensions respecting max constraints
	width := image.Width
	height := image.Height

	if maxWidth > 0 && width > maxWidth {
		ratio := float64(maxWidth) / float64(width)
		width = maxWidth
		height = int(float64(height) * ratio)
	}
	if maxHeight > 0 && height > maxHeight {
		ratio := float64(maxHeight) / float64(height)
		height = maxHeight
		width = int(float64(width) * ratio)
	}

	// Select appropriate variant URL based on size
	imageURL := h.selectVariantURL(image, width)
	thumbnailURL := h.selectThumbnailURL(image)
	thumbnailWidth, thumbnailHeight := h.getThumbnailDimensions(image)

	return OEmbedResponse{
		Type:            "photo",
		Version:         "1.0",
		URL:             imageURL,
		Width:           width,
		Height:          height,
		Title:           image.Title,
		AuthorName:      "", // Username not available in ImageDTO, could be fetched separately
		AuthorURL:       fmt.Sprintf("%s/users/%s", h.baseURL, image.OwnerID),
		ProviderName:    "goimg",
		ProviderURL:     h.baseURL,
		CacheAge:        3600, // 1 hour cache
		ThumbnailURL:    thumbnailURL,
		ThumbnailWidth:  thumbnailWidth,
		ThumbnailHeight: thumbnailHeight,
	}
}

// selectVariantURL selects the best image variant URL for the requested width.
func (h *OEmbedHandler) selectVariantURL(image *queries.ImageDTO, targetWidth int) string {
	// Find best matching variant
	// Variant sizes: thumbnail (150), small (320), medium (800), large (1600), original
	if targetWidth <= 150 {
		return h.getVariantURL(image, "thumbnail")
	} else if targetWidth <= 320 {
		return h.getVariantURL(image, "small")
	} else if targetWidth <= 800 {
		return h.getVariantURL(image, "medium")
	} else if targetWidth <= 1600 {
		return h.getVariantURL(image, "large")
	}
	return h.getVariantURL(image, "original")
}

// getVariantURL returns the URL for a specific variant.
func (h *OEmbedHandler) getVariantURL(image *queries.ImageDTO, variant string) string {
	// Check if variant exists in the image's variants
	for _, v := range image.Variants {
		if v.Type == variant {
			return fmt.Sprintf("%s/api/v1/images/%s/variants/%s", h.baseURL, image.ID, variant)
		}
	}
	// Fallback to original
	return fmt.Sprintf("%s/api/v1/images/%s/variants/original", h.baseURL, image.ID)
}

// selectThumbnailURL returns the thumbnail URL for the image.
func (h *OEmbedHandler) selectThumbnailURL(image *queries.ImageDTO) string {
	return fmt.Sprintf("%s/api/v1/images/%s/variants/thumbnail", h.baseURL, image.ID)
}

// getThumbnailDimensions returns the thumbnail dimensions.
func (h *OEmbedHandler) getThumbnailDimensions(image *queries.ImageDTO) (width, height int) {
	// Default thumbnail is 150px max width, maintaining aspect ratio
	for _, v := range image.Variants {
		if v.Type == "thumbnail" {
			return v.Width, v.Height
		}
	}
	// Fallback: calculate based on original dimensions
	if image.Width > image.Height {
		return 150, int(150 * float64(image.Height) / float64(image.Width))
	}
	return int(150 * float64(image.Width) / float64(image.Height)), 150
}

// writeJSONResponse writes the oEmbed response as JSON.
func (h *OEmbedHandler) writeJSONResponse(w http.ResponseWriter, response OEmbedResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode oEmbed JSON response")
	}
}

// writeXMLResponse writes the oEmbed response as XML.
func (h *OEmbedHandler) writeXMLResponse(w http.ResponseWriter, response OEmbedResponse) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	xmlResponse := OEmbedXMLResponse{OEmbedResponse: response}
	output, err := xml.MarshalIndent(xmlResponse, "", "  ")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode oEmbed XML response")
		return
	}

	// Write XML declaration and response
	w.Write([]byte(xml.Header))
	w.Write(output)
}

// writeError writes an error response.
func (h *OEmbedHandler) writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "xml" {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(status)
		w.Write([]byte(xml.Header))
		w.Write([]byte(fmt.Sprintf(`<oembed><error>%s</error></oembed>`, message)))
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
	}
}
