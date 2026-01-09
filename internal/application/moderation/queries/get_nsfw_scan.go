package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// GetNSFWScanQuery retrieves a single NSFW scan by its unique ID.
// This is a read-only operation with no side effects.
type GetNSFWScanQuery struct {
	ScanID string // Scan to retrieve
}

// Implement Query interface.
func (GetNSFWScanQuery) isQuery() {}

// NSFWScanDTO represents an NSFW scan data transfer object for HTTP responses.
type NSFWScanDTO struct {
	ID             string          `json:"id"`
	ImageID        string          `json:"image_id"`
	Provider       string          `json:"provider"`
	Status         string          `json:"status"`
	Category       string          `json:"category"`
	Score          float64         `json:"score"`
	IsNSFW         bool            `json:"is_nsfw"`
	RequiresReview bool            `json:"requires_review"`
	Details        *NSFWDetailsDTO `json:"details,omitempty"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	ScannedAt      *time.Time      `json:"scanned_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// NSFWDetailsDTO represents NSFW detection details.
type NSFWDetailsDTO struct {
	NudityScore    float64  `json:"nudity_score"`
	WeaponScore    float64  `json:"weapon_score"`
	ViolenceScore  float64  `json:"violence_score"`
	OffensiveScore float64  `json:"offensive_score"`
	DrugScore      float64  `json:"drug_score"`
	SubCategories  []string `json:"sub_categories,omitempty"`
}

// GetNSFWScanHandler processes GetNSFWScanQuery requests.
// It retrieves a single NSFW scan and converts it to a DTO.
type GetNSFWScanHandler struct {
	scans moderation.NSFWScanRepository
}

// NewGetNSFWScanHandler creates a new GetNSFWScanHandler with the given dependencies.
func NewGetNSFWScanHandler(scans moderation.NSFWScanRepository) *GetNSFWScanHandler {
	return &GetNSFWScanHandler{
		scans: scans,
	}
}

// Handle executes the GetNSFWScanQuery and returns the scan data.
//
// Returns:
//   - *NSFWScanDTO: The scan data
//   - error: ErrNSFWScanNotFound if the scan does not exist, or other repository errors
func (h *GetNSFWScanHandler) Handle(ctx context.Context, q GetNSFWScanQuery) (*NSFWScanDTO, error) {
	// Parse and validate scan ID.
	scanID, err := moderation.ParseNSFWScanID(q.ScanID)
	if err != nil {
		return nil, fmt.Errorf("invalid scan id: %w", err)
	}

	// Retrieve scan from repository.
	scan, err := h.scans.FindByID(ctx, scanID)
	if err != nil {
		return nil, fmt.Errorf("find scan by id: %w", err)
	}

	// Convert to DTO.
	return nsfwScanToDTO(scan), nil
}

// nsfwScanToDTO converts a domain NSFWScan to a NSFWScanDTO.
func nsfwScanToDTO(s *moderation.NSFWScan) *NSFWScanDTO {
	dto := &NSFWScanDTO{
		ID:             s.ID().String(),
		ImageID:        s.ImageID().String(),
		Provider:       s.Provider().String(),
		Status:         s.Status().String(),
		Category:       s.Category().String(),
		Score:          s.Score(),
		IsNSFW:         s.IsNSFW(),
		RequiresReview: s.RequiresReview(),
		ErrorMessage:   s.ErrorMessage(),
		CreatedAt:      s.CreatedAt(),
		UpdatedAt:      s.UpdatedAt(),
	}

	// Set scanned_at if the scan has completed.
	if !s.ScannedAt().IsZero() {
		scannedAt := s.ScannedAt()
		dto.ScannedAt = &scannedAt
	}

	// Add details if not empty.
	details := s.Details()
	if !details.IsEmpty() {
		dto.Details = &NSFWDetailsDTO{
			NudityScore:    details.NudityScore,
			WeaponScore:    details.WeaponScore,
			ViolenceScore:  details.ViolenceScore,
			OffensiveScore: details.OffensiveScore,
			DrugScore:      details.DrugScore,
			SubCategories:  details.SubCategories,
		}
	}

	return dto
}
