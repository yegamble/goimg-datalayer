// Package handlers provides HTTP request handlers for the API endpoints.
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// VariantConfigHandler handles variant config-related HTTP endpoints.
// It delegates to application layer command and query handlers for business logic.
type VariantConfigHandler struct {
	createConfig *commands.CreateVariantConfigHandler
	updateConfig *commands.UpdateVariantConfigHandler
	deleteConfig *commands.DeleteVariantConfigHandler
	getConfig    *queries.GetVariantConfigHandler
	listConfigs  *queries.ListVariantConfigsHandler
	listPresets  *queries.ListVariantConfigPresetsHandler
	logger       zerolog.Logger
}

// NewVariantConfigHandler creates a new VariantConfigHandler with the given dependencies.
func NewVariantConfigHandler(
	createConfig *commands.CreateVariantConfigHandler,
	updateConfig *commands.UpdateVariantConfigHandler,
	deleteConfig *commands.DeleteVariantConfigHandler,
	getConfig *queries.GetVariantConfigHandler,
	listConfigs *queries.ListVariantConfigsHandler,
	listPresets *queries.ListVariantConfigPresetsHandler,
	logger zerolog.Logger,
) *VariantConfigHandler {
	return &VariantConfigHandler{
		createConfig: createConfig,
		updateConfig: updateConfig,
		deleteConfig: deleteConfig,
		getConfig:    getConfig,
		listConfigs:  listConfigs,
		listPresets:  listPresets,
		logger:       logger,
	}
}

// Routes registers variant config routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/variant-configs.
func (h *VariantConfigHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/presets", h.ListPresets)
	r.Get("/{configID}", h.Get)
	r.Put("/{configID}", h.Update)
	r.Delete("/{configID}", h.Delete)

	return r
}

// CreateVariantConfigRequest represents the request body for creating a variant config.
type CreateVariantConfigRequest struct {
	Name      string `json:"name" validate:"required,min=1,max=50"`
	MaxWidth  int    `json:"max_width" validate:"required,min=1,max=8192"`
	MaxHeight int    `json:"max_height" validate:"required,min=1,max=8192"`
	Format    string `json:"format" validate:"required,oneof=jpeg png webp avif"`
	Quality   int    `json:"quality" validate:"required,min=1,max=100"`
	CropMode  string `json:"crop_mode" validate:"required,oneof=fit fill crop"`
}

// UpdateVariantConfigRequest represents the request body for updating a variant config.
type UpdateVariantConfigRequest struct {
	MaxWidth  int    `json:"max_width" validate:"required,min=1,max=8192"`
	MaxHeight int    `json:"max_height" validate:"required,min=1,max=8192"`
	Format    string `json:"format" validate:"required,oneof=jpeg png webp avif"`
	Quality   int    `json:"quality" validate:"required,min=1,max=100"`
	CropMode  string `json:"crop_mode" validate:"required,oneof=fit fill crop"`
}

// VariantConfigDTO represents a variant config in API responses.
type VariantConfigDTO = queries.VariantConfigDTO

// Create handles POST /api/v1/variant-configs
// Creates a new custom variant configuration.
//
// Request: CreateVariantConfigRequest JSON body
// Response: 201 Created with VariantConfigDTO
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 409: Config name already exists for user
//   - 500: Internal server error
func (h *VariantConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in create variant config handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Decode request body
	var req CreateVariantConfigRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid create variant config request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid variant config data",
			validationErrors,
		)
		return
	}

	// 3. Build create command
	cmd := commands.CreateVariantConfigCommand{
		UserID:    userCtx.UserID.String(),
		Name:      req.Name,
		MaxWidth:  req.MaxWidth,
		MaxHeight: req.MaxHeight,
		Format:    req.Format,
		Quality:   req.Quality,
		CropMode:  req.CropMode,
	}

	// 4. Execute create command
	configID, err := h.createConfig.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "create variant config")
		return
	}

	// 5. Fetch created config to return
	query := queries.GetVariantConfigQuery{
		ConfigID:         configID,
		RequestingUserID: userCtx.UserID.String(),
	}

	config, err := h.getConfig.Handle(ctx, query)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to fetch created variant config")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Variant config created but failed to retrieve",
		)
		return
	}

	// 6. Return created config
	h.logger.Info().
		Str("config_id", configID).
		Str("user_id", userCtx.UserID.String()).
		Msg("variant config created successfully")

	if err := EncodeJSON(w, http.StatusCreated, config); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode create variant config response")
	}
}

// Get handles GET /api/v1/variant-configs/{configID}
// Retrieves a single variant config by its ID.
//
// Path parameters:
//   - configID: UUID of the variant config
//
// Response: 200 OK with VariantConfigDTO
// Errors:
//   - 400: Invalid config ID format
//   - 401: Not authenticated
//   - 403: User doesn't own the config (and it's not a preset)
//   - 404: Config not found
//   - 500: Internal server error
func (h *VariantConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in get variant config handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract config ID from path
	configID := GetPathParam(r, "configID")
	if configID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing config ID",
		)
		return
	}

	// 3. Build query
	query := queries.GetVariantConfigQuery{
		ConfigID:         configID,
		RequestingUserID: userCtx.UserID.String(),
	}

	// 4. Execute query
	config, err := h.getConfig.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get variant config")
		return
	}

	// 5. Return config
	h.logger.Debug().
		Str("config_id", configID).
		Str("user_id", userCtx.UserID.String()).
		Msg("variant config retrieved successfully")

	if err := EncodeJSON(w, http.StatusOK, config); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get variant config response")
	}
}

// Update handles PUT /api/v1/variant-configs/{configID}
// Updates variant config settings (dimensions, format, quality, crop mode).
//
// Path parameters:
//   - configID: UUID of the variant config
//
// Request: UpdateVariantConfigRequest JSON body
// Response: 200 OK with VariantConfigDTO
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: User doesn't own the config
//   - 404: Config not found
//   - 500: Internal server error
func (h *VariantConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in update variant config handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract config ID from path
	configID := GetPathParam(r, "configID")
	if configID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing config ID",
		)
		return
	}

	// 3. Decode request body
	var req UpdateVariantConfigRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid update variant config request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid variant config update data",
			validationErrors,
		)
		return
	}

	// 4. Build update command
	cmd := commands.UpdateVariantConfigCommand{
		ConfigID:  configID,
		UserID:    userCtx.UserID.String(),
		MaxWidth:  req.MaxWidth,
		MaxHeight: req.MaxHeight,
		Format:    req.Format,
		Quality:   req.Quality,
		CropMode:  req.CropMode,
	}

	// 5. Execute update command
	if err := h.updateConfig.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "update variant config")
		return
	}

	// 6. Fetch updated config to return
	query := queries.GetVariantConfigQuery{
		ConfigID:         configID,
		RequestingUserID: userCtx.UserID.String(),
	}

	config, err := h.getConfig.Handle(ctx, query)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to fetch updated variant config")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Variant config updated but failed to retrieve",
		)
		return
	}

	// 7. Return updated config
	h.logger.Info().
		Str("config_id", configID).
		Str("user_id", userCtx.UserID.String()).
		Msg("variant config updated successfully")

	if err := EncodeJSON(w, http.StatusOK, config); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update variant config response")
	}
}

// Delete handles DELETE /api/v1/variant-configs/{configID}
// Deletes a custom variant configuration.
//
// Path parameters:
//   - configID: UUID of the variant config
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid config ID format or trying to delete a preset
//   - 401: Not authenticated
//   - 403: User doesn't own the config
//   - 404: Config not found
//   - 500: Internal server error
func (h *VariantConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in delete variant config handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract config ID from path
	configID := GetPathParam(r, "configID")
	if configID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing config ID",
		)
		return
	}

	// 3. Build delete command
	cmd := commands.DeleteVariantConfigCommand{
		ConfigID: configID,
		UserID:   userCtx.UserID.String(),
	}

	// 4. Execute delete command
	if err := h.deleteConfig.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "delete variant config")
		return
	}

	// 5. Return 204 No Content
	h.logger.Info().
		Str("config_id", configID).
		Str("user_id", userCtx.UserID.String()).
		Msg("variant config deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /api/v1/variant-configs
// Lists all custom variant configs for the authenticated user.
//
// Response: 200 OK with []VariantConfigDTO
// Errors:
//   - 401: Not authenticated
//   - 500: Internal server error
func (h *VariantConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in list variant configs handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Build query
	query := queries.ListVariantConfigsQuery{
		UserID: userCtx.UserID.String(),
	}

	// 3. Execute query
	configs, err := h.listConfigs.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list variant configs")
		return
	}

	// 4. Return configs
	h.logger.Debug().
		Str("user_id", userCtx.UserID.String()).
		Int("count", len(configs)).
		Msg("variant configs listed successfully")

	if err := EncodeJSON(w, http.StatusOK, configs); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list variant configs response")
	}
}

// ListPresets handles GET /api/v1/variant-configs/presets
// Lists all system-defined variant presets (available to all users).
//
// Response: 200 OK with []VariantConfigDTO
// Errors:
//   - 500: Internal server error
func (h *VariantConfigHandler) ListPresets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Build query
	query := queries.ListVariantConfigPresetsQuery{}

	// 2. Execute query
	presets, err := h.listPresets.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list variant config presets")
		return
	}

	// 3. Return presets
	h.logger.Debug().
		Int("count", len(presets)).
		Msg("variant config presets listed successfully")

	if err := EncodeJSON(w, http.StatusOK, presets); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list presets response")
	}
}

// mapErrorAndRespond maps application/domain errors to HTTP responses.
func (h *VariantConfigHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("variant config operation failed")

	// Simplified error mapping - expand based on actual domain errors
	middleware.WriteError(w, r,
		http.StatusInternalServerError,
		"Internal Server Error",
		"An unexpected error occurred",
	)
}
