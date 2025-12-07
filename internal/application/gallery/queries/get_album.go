package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GetAlbumQuery represents a query to retrieve album details.
type GetAlbumQuery struct {
	AlbumID          string
	RequestingUserID string // Optional: empty string if not authenticated
}

// AlbumDTO represents the album data transfer object.
type AlbumDTO struct {
	ID           string  `json:"id"`
	OwnerID      string  `json:"owner_id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Visibility   string  `json:"visibility"`
	CoverImageID *string `json:"cover_image_id,omitempty"`
	ImageCount   int     `json:"image_count"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// GetAlbumHandler processes get album queries.
// It retrieves album details and respects visibility settings.
type GetAlbumHandler struct {
	albums gallery.AlbumRepository
}

// NewGetAlbumHandler creates a new GetAlbumHandler.
func NewGetAlbumHandler(albums gallery.AlbumRepository) *GetAlbumHandler {
	return &GetAlbumHandler{
		albums: albums,
	}
}

// Handle executes the get album query.
//
// Process flow:
//  1. Parse album ID
//  2. Retrieve album from repository
//  3. Check visibility permissions
//  4. Convert to DTO and return
//
// Returns:
//   - AlbumDTO on success
//   - ErrAlbumNotFound if album doesn't exist
//   - Authorization error if requesting user cannot view the album
func (h *GetAlbumHandler) Handle(ctx context.Context, q GetAlbumQuery) (*AlbumDTO, error) {
	// 1. Parse album ID
	albumID, err := gallery.ParseAlbumID(q.AlbumID)
	if err != nil {
		return nil, fmt.Errorf("invalid album id: %w", err)
	}

	// 2. Retrieve album
	album, err := h.albums.FindByID(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("find album: %w", err)
	}

	// 3. Check visibility permissions
	// Parse requesting user ID (may be empty for anonymous users)
	var requestingUserID identity.UserID
	if q.RequestingUserID != "" {
		requestingUserID, err = identity.ParseUserID(q.RequestingUserID)
		if err != nil {
			return nil, fmt.Errorf("invalid requesting user id: %w", err)
		}
	}

	// Check if user can view this album based on visibility
	if !canViewAlbum(album, requestingUserID) {
		return nil, fmt.Errorf("unauthorized: cannot view this album")
	}

	// 4. Convert to DTO
	dto := albumToDTO(album)
	return &dto, nil
}

// canViewAlbum checks if a user can view an album based on visibility settings.
func canViewAlbum(album *gallery.Album, requestingUserID identity.UserID) bool {
	// Public albums can be viewed by anyone
	if album.Visibility() == gallery.VisibilityPublic {
		return true
	}

	// Private/unlisted albums can only be viewed by the owner
	// Empty/zero requestingUserID means anonymous user
	if requestingUserID.IsZero() {
		return false
	}

	return album.IsOwnedBy(requestingUserID)
}

// albumToDTO converts a domain Album to a DTO.
func albumToDTO(album *gallery.Album) AlbumDTO {
	var coverImageID *string
	if album.CoverImageID() != nil {
		id := album.CoverImageID().String()
		coverImageID = &id
	}

	return AlbumDTO{
		ID:           album.ID().String(),
		OwnerID:      album.OwnerID().String(),
		Title:        album.Title(),
		Description:  album.Description(),
		Visibility:   album.Visibility().String(),
		CoverImageID: coverImageID,
		ImageCount:   album.ImageCount(),
		CreatedAt:    album.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    album.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}
