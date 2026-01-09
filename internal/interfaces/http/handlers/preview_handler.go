package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
)

// previewHTMLTemplate is the inline HTML template for image preview pages.
// Includes Open Graph and Twitter Card meta tags for social media sharing.
const previewHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - goimg</title>

    <!-- Open Graph (Facebook, LinkedIn) -->
    <meta property="og:title" content="{{.Title}}">
    <meta property="og:description" content="{{.Description}}">
    <meta property="og:image" content="{{.ImageURL}}">
    <meta property="og:url" content="{{.PageURL}}">
    <meta property="og:type" content="article">
    <meta property="og:site_name" content="goimg">
    <meta property="og:image:width" content="{{.Width}}">
    <meta property="og:image:height" content="{{.Height}}">

    <!-- Twitter Card -->
    <meta name="twitter:card" content="summary_large_image">
    <meta name="twitter:title" content="{{.Title}}">
    <meta name="twitter:description" content="{{.Description}}">
    <meta name="twitter:image" content="{{.ImageURL}}">
    <meta name="twitter:image:alt" content="{{.Title}}">

    <!-- Additional SEO -->
    <meta name="description" content="{{.Description}}">
    <link rel="canonical" href="{{.PageURL}}">

    <!-- oEmbed discovery -->
    <link rel="alternate" type="application/json+oembed" href="{{.OEmbedURL}}" title="{{.Title}}">

    <style>
        body {
            margin: 0;
            padding: 20px;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background: #f5f5f5;
            color: #333;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 40px;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        h1 {
            margin: 0 0 16px 0;
            font-size: 32px;
            font-weight: 600;
        }
        .description {
            color: #666;
            margin-bottom: 32px;
            line-height: 1.6;
        }
        .image-wrapper {
            text-align: center;
            margin-bottom: 32px;
        }
        img {
            max-width: 100%;
            height: auto;
            border-radius: 4px;
        }
        .meta {
            color: #999;
            font-size: 14px;
        }
        .stats {
            display: flex;
            gap: 24px;
            margin-top: 16px;
        }
        .stat {
            display: flex;
            align-items: center;
            gap: 4px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{.Title}}</h1>
        {{if .Description}}
        <p class="description">{{.Description}}</p>
        {{end}}
        <div class="image-wrapper">
            <img src="{{.ImageURL}}" alt="{{.Title}}" loading="lazy">
        </div>
        <div class="meta">
            <p>{{.Width}} x {{.Height}} pixels</p>
            <div class="stats">
                <span class="stat">{{.ViewCount}} views</span>
                <span class="stat">{{.LikeCount}} likes</span>
            </div>
        </div>
    </div>
</body>
</html>`

// PreviewData holds data for rendering the preview template.
type PreviewData struct {
	Title       string
	Description string
	ImageURL    string // Absolute URL to medium/large variant
	PageURL     string // Canonical URL to this preview page
	OEmbedURL   string // URL to oEmbed endpoint for this image
	Width       int
	Height      int
	ViewCount   int64
	LikeCount   int64
}

// PreviewHandler handles public image preview pages with social meta tags.
// Serves SEO-friendly HTML pages for social media link previews.
type PreviewHandler struct {
	getImage *queries.GetImageHandler
	baseURL  string             // Base URL for generating absolute URLs
	template *template.Template // Pre-compiled HTML template
	logger   zerolog.Logger
}

// NewPreviewHandler creates a new preview handler.
func NewPreviewHandler(
	getImage *queries.GetImageHandler,
	baseURL string,
	logger zerolog.Logger,
) *PreviewHandler {
	tmpl := template.Must(template.New("preview").Parse(previewHTMLTemplate))

	return &PreviewHandler{
		getImage: getImage,
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		template: tmpl,
		logger:   logger,
	}
}

// Routes returns a chi.Router with preview routes.
// Mount this at /images (not under /api/v1)
func (h *PreviewHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{imageID}/preview", h.GetPreview)
	return r
}

// GetPreview handles GET /images/{imageID}/preview
// Returns an HTML page with Open Graph and Twitter Card meta tags.
//
// Process flow:
//  1. Extract image ID from URL path
//  2. Fetch image details via GetImageHandler
//  3. Validate image is public (return 404 for private)
//  4. Select appropriate variant URL (medium or large)
//  5. Render HTML template with meta tags
//  6. Set proper Content-Type and cache headers
func (h *PreviewHandler) GetPreview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract image ID from URL path
	imageID := chi.URLParam(r, "imageID")
	if imageID == "" {
		h.logger.Warn().Msg("missing image ID in preview request")
		http.NotFound(w, r)
		return
	}

	// 2. Fetch image details (no user context = public images only)
	query := queries.GetImageQuery{
		ImageID: imageID,
		// No RequestingUserID = only public images returned
	}

	image, err := h.getImage.Handle(ctx, query)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID).
			Msg("image not found for preview")
		http.NotFound(w, r)
		return
	}

	// 3. Validate image is public
	if image.Visibility != "public" {
		h.logger.Debug().
			Str("image_id", imageID).
			Str("visibility", image.Visibility).
			Msg("non-public image requested for preview")
		http.NotFound(w, r)
		return
	}

	// 4. Build preview data
	data := h.buildPreviewData(image)

	// 5. Set headers
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Cache headers for social media crawlers
	w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour

	// Security headers (redundant with middleware but safe)
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	w.WriteHeader(http.StatusOK)

	// 6. Render template
	if err := h.template.Execute(w, data); err != nil {
		h.logger.Error().Err(err).Msg("failed to render preview template")
		return
	}

	h.logger.Debug().
		Str("image_id", imageID).
		Str("title", image.Title).
		Msg("preview page rendered successfully")
}

// buildPreviewData constructs template data from ImageDTO.
func (h *PreviewHandler) buildPreviewData(image *queries.ImageDTO) PreviewData {
	// Select best variant for social media preview (medium or large)
	imageURL := h.selectVariantURL(image, 1200) // Target 1200px width for social previews

	// Build canonical URL to this preview page
	pageURL := fmt.Sprintf("%s/images/%s/preview", h.baseURL, image.ID)

	// Build oEmbed URL for discovery
	oembedURL := fmt.Sprintf("%s/api/v1/oembed?url=%s", h.baseURL, pageURL)

	// Sanitize description for meta tags
	description := image.Description
	if description == "" {
		if image.Title != "" {
			description = fmt.Sprintf("View %s on goimg", image.Title)
		} else {
			description = "View this image on goimg"
		}
	}
	if len(description) > 200 {
		description = description[:197] + "..."
	}

	// Use title or fallback
	title := image.Title
	if title == "" {
		title = "Untitled Image"
	}

	return PreviewData{
		Title:       title,
		Description: description,
		ImageURL:    imageURL,
		PageURL:     pageURL,
		OEmbedURL:   oembedURL,
		Width:       image.Width,
		Height:      image.Height,
		ViewCount:   image.ViewCount,
		LikeCount:   image.LikeCount,
	}
}

// selectVariantURL chooses the best image variant for social media previews.
func (h *PreviewHandler) selectVariantURL(image *queries.ImageDTO, targetWidth int) string {
	// Find best matching variant based on target width
	// Variants: thumbnail (150), small (320), medium (800), large (1600), original
	variantType := "medium" // Default for social media (800px)

	if targetWidth <= 320 {
		variantType = "small"
	} else if targetWidth > 800 {
		variantType = "large"
	}

	// Check if variant exists
	for _, v := range image.Variants {
		if v.Type == variantType {
			return fmt.Sprintf("%s/api/v1/images/%s/variants/%s", h.baseURL, image.ID, variantType)
		}
	}

	// Fallback to original
	return fmt.Sprintf("%s/api/v1/images/%s/variants/original", h.baseURL, image.ID)
}

// formatUploadDate formats a timestamp string into a human-readable date.
func formatUploadDate(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return t.Format("January 2, 2006")
}
