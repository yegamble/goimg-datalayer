package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// VariantConfigDTO represents the variant config data transfer object.
type VariantConfigDTO struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	MaxWidth  int    `json:"max_width"`
	MaxHeight int    `json:"max_height"`
	Format    string `json:"format"`
	Quality   int    `json:"quality"`
	CropMode  string `json:"crop_mode"`
	IsPreset  bool   `json:"is_preset"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// GetVariantConfigQuery represents a query to retrieve a variant config.
type GetVariantConfigQuery struct {
	ConfigID         string
	RequestingUserID string // Required: user must own the config or it must be a preset
}

// GetVariantConfigHandler processes get variant config queries.
type GetVariantConfigHandler struct {
	configs gallery.VariantConfigRepository
}

// NewGetVariantConfigHandler creates a new GetVariantConfigHandler.
func NewGetVariantConfigHandler(configs gallery.VariantConfigRepository) *GetVariantConfigHandler {
	return &GetVariantConfigHandler{
		configs: configs,
	}
}

// Handle executes the get variant config query.
//
// Process flow:
//  1. Parse config ID and user ID
//  2. Retrieve config from repository
//  3. Check access permissions (owner or preset)
//  4. Convert to DTO and return
//
// Returns:
//   - VariantConfigDTO on success
//   - ErrVariantConfigNotFound if config doesn't exist
//   - Authorization error if user cannot access the config
func (h *GetVariantConfigHandler) Handle(ctx context.Context, q GetVariantConfigQuery) (*VariantConfigDTO, error) {
	// 1. Parse config ID
	configID, err := gallery.ParseVariantConfigID(q.ConfigID)
	if err != nil {
		return nil, fmt.Errorf("invalid config id: %w", err)
	}

	// 2. Parse requesting user ID
	requestingUserID, err := identity.ParseUserID(q.RequestingUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid requesting user id: %w", err)
	}

	// 3. Retrieve config
	config, err := h.configs.FindByID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("find variant config: %w", err)
	}

	// 4. Check access permissions
	if !config.IsPreset() && !config.IsOwnedBy(requestingUserID) {
		return nil, fmt.Errorf("unauthorized: cannot access this variant config")
	}

	// 5. Convert to DTO
	dto := variantConfigToDTO(config)
	return &dto, nil
}

// ListVariantConfigsQuery represents a query to list a user's variant configs.
type ListVariantConfigsQuery struct {
	UserID string
}

// ListVariantConfigsHandler processes list variant configs queries.
type ListVariantConfigsHandler struct {
	configs gallery.VariantConfigRepository
}

// NewListVariantConfigsHandler creates a new ListVariantConfigsHandler.
func NewListVariantConfigsHandler(configs gallery.VariantConfigRepository) *ListVariantConfigsHandler {
	return &ListVariantConfigsHandler{
		configs: configs,
	}
}

// Handle executes the list variant configs query.
//
// Returns:
//   - []VariantConfigDTO on success (user's custom configs)
//   - Validation error if user ID is invalid
func (h *ListVariantConfigsHandler) Handle(ctx context.Context, q ListVariantConfigsQuery) ([]VariantConfigDTO, error) {
	// 1. Parse user ID
	userID, err := identity.ParseUserID(q.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Retrieve configs
	configs, err := h.configs.FindByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find variant configs: %w", err)
	}

	// 3. Convert to DTOs
	dtos := make([]VariantConfigDTO, len(configs))
	for i, config := range configs {
		dtos[i] = variantConfigToDTO(config)
	}

	return dtos, nil
}

// ListVariantConfigPresetsQuery represents a query to list system variant presets.
type ListVariantConfigPresetsQuery struct{}

// ListVariantConfigPresetsHandler processes list variant config presets queries.
type ListVariantConfigPresetsHandler struct {
	configs gallery.VariantConfigRepository
}

// NewListVariantConfigPresetsHandler creates a new ListVariantConfigPresetsHandler.
func NewListVariantConfigPresetsHandler(configs gallery.VariantConfigRepository) *ListVariantConfigPresetsHandler {
	return &ListVariantConfigPresetsHandler{
		configs: configs,
	}
}

// Handle executes the list variant config presets query.
//
// Returns:
//   - []VariantConfigDTO on success (system-defined presets)
func (h *ListVariantConfigPresetsHandler) Handle(ctx context.Context, _ ListVariantConfigPresetsQuery) ([]VariantConfigDTO, error) {
	// 1. Retrieve presets
	presets, err := h.configs.FindPresets(ctx)
	if err != nil {
		return nil, fmt.Errorf("find variant config presets: %w", err)
	}

	// 2. Convert to DTOs
	dtos := make([]VariantConfigDTO, len(presets))
	for i, config := range presets {
		dtos[i] = variantConfigToDTO(config)
	}

	return dtos, nil
}

// variantConfigToDTO converts a domain VariantConfig to a DTO.
func variantConfigToDTO(config *gallery.VariantConfig) VariantConfigDTO {
	return VariantConfigDTO{
		ID:        config.ID().String(),
		UserID:    config.UserID().String(),
		Name:      config.Name(),
		MaxWidth:  config.MaxWidth(),
		MaxHeight: config.MaxHeight(),
		Format:    string(config.Format()),
		Quality:   config.Quality(),
		CropMode:  string(config.CropMode()),
		IsPreset:  config.IsPreset(),
		CreatedAt: config.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: config.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}
