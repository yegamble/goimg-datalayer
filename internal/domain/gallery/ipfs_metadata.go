package gallery

import (
	"fmt"
	"strings"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// CID length constraints for validation.
const (
	cidV0Length    = 46 // CIDv0: Qm... (base58btc, 46 characters)
	cidV1MinLength = 50 // CIDv1: bafy... (base32, minimum 50 characters)
	cidOtherMinLen = 10 // Other CIDv1 bases: b... (minimum length)
)

// IPFSMetadata is a value object containing IPFS storage information.
// It is immutable after creation and optional (images may not have IPFS backup).
type IPFSMetadata struct {
	cid      string     // Content Identifier (CID)
	pinned   bool       // Whether content is pinned to IPFS node
	pinnedAt *time.Time // When the content was pinned (nil if not pinned)
}

// NewIPFSMetadata creates a new IPFSMetadata with validation.
// The CID is required and must be a valid IPFS Content Identifier.
func NewIPFSMetadata(cid string, pinned bool, pinnedAt *time.Time) (IPFSMetadata, error) {
	cid = strings.TrimSpace(cid)
	if err := validateCID(cid); err != nil {
		return IPFSMetadata{}, err
	}

	// pinnedAt should only be set if pinned is true
	if !pinned {
		pinnedAt = nil
	}

	return IPFSMetadata{
		cid:      cid,
		pinned:   pinned,
		pinnedAt: pinnedAt,
	}, nil
}

// validateCID checks if a string is a valid IPFS CID.
// Supports both CIDv0 (Qm...) and CIDv1 (bafy...).
func validateCID(cid string) error {
	if cid == "" {
		return fmt.Errorf("%w: empty CID", shared.ErrInvalidInput)
	}

	// CIDv0: starts with "Qm", base58btc encoded, 46 characters
	if strings.HasPrefix(cid, "Qm") {
		if len(cid) != cidV0Length {
			return fmt.Errorf("%w: invalid CIDv0 length", shared.ErrInvalidInput)
		}
		return nil
	}

	// CIDv1: starts with "bafy" (base32) or "bafk" (raw)
	if strings.HasPrefix(cid, "bafy") || strings.HasPrefix(cid, "bafk") {
		if len(cid) < cidV1MinLength {
			return fmt.Errorf("%w: invalid CIDv1 length", shared.ErrInvalidInput)
		}
		return nil
	}

	// Allow other CIDv1 formats starting with 'b'
	if strings.HasPrefix(cid, "b") && len(cid) > cidOtherMinLen {
		return nil
	}

	return fmt.Errorf("%w: unrecognized CID format", shared.ErrInvalidInput)
}

// CID returns the IPFS Content Identifier.
func (m IPFSMetadata) CID() string {
	return m.cid
}

// Pinned returns whether the content is pinned to the IPFS node.
func (m IPFSMetadata) Pinned() bool {
	return m.pinned
}

// PinnedAt returns when the content was pinned (nil if not pinned).
func (m IPFSMetadata) PinnedAt() *time.Time {
	return m.pinnedAt
}

// IsZero returns true if this is an empty/unset IPFSMetadata.
func (m IPFSMetadata) IsZero() bool {
	return m.cid == ""
}

// URI returns the canonical IPFS URI (ipfs://<cid>).
func (m IPFSMetadata) URI() string {
	if m.cid == "" {
		return ""
	}
	return fmt.Sprintf("ipfs://%s", m.cid)
}

// GatewayURL returns the public gateway URL for the content.
// The gateway parameter should be the gateway base URL (e.g., "https://ipfs.io").
func (m IPFSMetadata) GatewayURL(gateway string) string {
	if m.cid == "" || gateway == "" {
		return ""
	}
	return fmt.Sprintf("%s/ipfs/%s", strings.TrimSuffix(gateway, "/"), m.cid)
}

// WithPinned returns a new IPFSMetadata with the pinned status updated.
func (m IPFSMetadata) WithPinned(pinned bool) IPFSMetadata {
	var pinnedAt *time.Time
	if pinned {
		now := time.Now().UTC()
		pinnedAt = &now
	}
	return IPFSMetadata{
		cid:      m.cid,
		pinned:   pinned,
		pinnedAt: pinnedAt,
	}
}

// Equals returns true if two IPFSMetadata values have the same CID.
// Pin status is not considered for equality.
func (m IPFSMetadata) Equals(other IPFSMetadata) bool {
	return m.cid == other.cid
}
