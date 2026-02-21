package handlers

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

type ImageHandler struct {
	uploadImage           *commands.UploadImageHandler
	updateImage           *commands.UpdateImageHandler
	deleteImage           *commands.DeleteImageHandler
	generateCustomVariant *commands.GenerateCustomVariantHandler
	getImage              *queries.GetImageHandler
	listImages            *queries.ListImagesHandler
	searchImages          *queries.SearchImagesHandler
	storage               StorageProvider
	baseURL               string
	logger                zerolog.Logger
}

type StorageProvider interface {
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

func NewImageHandler(
	uploadImage *commands.UploadImageHandler,
	updateImage *commands.UpdateImageHandler,
	deleteImage *commands.DeleteImageHandler,
	generateCustomVariant *commands.GenerateCustomVariantHandler,
	getImage *queries.GetImageHandler,
	listImages *queries.ListImagesHandler,
	searchImages *queries.SearchImagesHandler,
	storage StorageProvider,
	baseURL string,
	logger zerolog.Logger,
) *ImageHandler {
	return &ImageHandler{
		uploadImage:           uploadImage,
		updateImage:           updateImage,
		deleteImage:           deleteImage,
		generateCustomVariant: generateCustomVariant,
		getImage:              getImage,
		listImages:            listImages,
		searchImages:          searchImages,
		storage:               storage,
		baseURL:               strings.TrimRight(baseURL, "/"),
		logger:                logger,
	}
}

// Note: Authentication and rate limiting middleware should be applied at the router level.
// Note: The variant endpoint (/{imageID}/variants/{size}) is registered in the image router.
func (h *ImageHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Upload)
	r.Get("/", h.List)
	r.Get("/search", h.Search)
	r.Get("/{imageID}", h.Get)
	r.Get("/{imageID}/download", h.Download)
	r.Put("/{imageID}", h.Update)
	r.Delete("/{imageID}", h.Delete)

	if h.generateCustomVariant != nil {
		r.Post("/{imageID}/variants", h.GenerateCustomVariant)
	}

	return r
}

func (h *ImageHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("image operation failed")

	middleware.WriteError(w, r,
		http.StatusInternalServerError,
		"Internal Server Error",
		"An unexpected error occurred",
	)
}
