package testhelpers

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// Test constants.
const (
	ValidUserID      = "550e8400-e29b-41d4-a716-446655440000"
	ValidAdminID     = "550e8400-e29b-41d4-a716-446655440001"
	ValidModeratorID = "550e8400-e29b-41d4-a716-446655440002"
	ValidImageID     = "7c9e6679-7425-40de-944b-e07fc1f90ae7"
	ValidReportID    = "8d0e6679-7425-40de-944b-e07fc1f90ae8"
	ValidBanID       = "9e1e6679-7425-40de-944b-e07fc1f90ae9"
	ValidReviewID    = "ae2e6679-7425-40de-944b-e07fc1f90aea"

	ValidBanReason     = "Violation of community guidelines"
	ValidReportReason  = "spam"
	ValidReportDesc    = "This image contains spam content"
	ValidReviewComment = "Reviewed and confirmed violation"
)

// TestSuite provides mock dependencies for moderation tests.
type TestSuite struct {
	BanRepo        *MockBanRepository
	ReportRepo     *MockReportRepository
	ReviewRepo     *MockReviewRepository
	NSFWScanRepo   *MockNSFWScanRepository
	ImageRepo      *MockImageRepository
	EventPublisher *MockEventPublisher
	Logger         zerolog.Logger
}

// NewTestSuite creates a new test suite with mocked dependencies.
func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	return &TestSuite{
		BanRepo:        new(MockBanRepository),
		ReportRepo:     new(MockReportRepository),
		ReviewRepo:     new(MockReviewRepository),
		NSFWScanRepo:   new(MockNSFWScanRepository),
		ImageRepo:      new(MockImageRepository),
		EventPublisher: new(MockEventPublisher),
		Logger:         zerolog.Nop(), // No-op logger for tests
	}
}

// AssertExpectations verifies all mock expectations were met.
func (s *TestSuite) AssertExpectations(t *testing.T) {
	t.Helper()

	s.BanRepo.AssertExpectations(t)
	s.ReportRepo.AssertExpectations(t)
	s.ReviewRepo.AssertExpectations(t)
	s.NSFWScanRepo.AssertExpectations(t)
	s.ImageRepo.AssertExpectations(t)
	s.EventPublisher.AssertExpectations(t)
}

// ValidUserIDParsed returns a parsed UserID for testing.
func ValidUserIDParsed() identity.UserID {
	userID, _ := identity.ParseUserID(ValidUserID)
	return userID
}

// ValidAdminIDParsed returns a parsed admin UserID for testing.
func ValidAdminIDParsed() identity.UserID {
	userID, _ := identity.ParseUserID(ValidAdminID)
	return userID
}

// ValidModeratorIDParsed returns a parsed moderator UserID for testing.
func ValidModeratorIDParsed() identity.UserID {
	userID, _ := identity.ParseUserID(ValidModeratorID)
	return userID
}

// ValidImageIDParsed returns a parsed ImageID for testing.
func ValidImageIDParsed() gallery.ImageID {
	imageID, _ := gallery.ParseImageID(ValidImageID)
	return imageID
}

// ValidReportIDParsed returns a parsed ReportID for testing.
func ValidReportIDParsed() moderation.ReportID {
	reportID, _ := moderation.ParseReportID(ValidReportID)
	return reportID
}

// ValidBanIDParsed returns a parsed BanID for testing.
func ValidBanIDParsed() moderation.BanID {
	banID, _ := moderation.ParseBanID(ValidBanID)
	return banID
}

// ValidBan creates a valid Ban aggregate for testing.
func ValidBan(t *testing.T) *moderation.Ban {
	t.Helper()

	userID := ValidUserIDParsed()
	adminID := ValidAdminIDParsed()

	ban, err := moderation.NewBan(userID, adminID, ValidBanReason, nil)
	require.NoError(t, err)
	ban.ClearEvents() // Clear events for testing

	return ban
}

// ValidTemporaryBan creates a valid temporary Ban aggregate for testing.
func ValidTemporaryBan(t *testing.T, duration time.Duration) *moderation.Ban {
	t.Helper()

	userID := ValidUserIDParsed()
	adminID := ValidAdminIDParsed()

	ban, err := moderation.NewBan(userID, adminID, ValidBanReason, &duration)
	require.NoError(t, err)
	ban.ClearEvents() // Clear events for testing

	return ban
}

// ValidReport creates a valid Report aggregate for testing.
func ValidReport(t *testing.T) *moderation.Report {
	t.Helper()

	reporterID := ValidUserIDParsed()
	imageID := ValidImageIDParsed()
	reason, _ := moderation.ParseReportReason(ValidReportReason)

	report, err := moderation.NewReport(reporterID, imageID, reason, ValidReportDesc)
	require.NoError(t, err)
	report.ClearEvents() // Clear events for testing

	return report
}

// ValidImage creates a valid Image aggregate for testing.
// The image owner is different from the reporter (ValidAdminID) to allow reporting.
func ValidImage(t *testing.T) *gallery.Image {
	t.Helper()

	ownerID := ValidAdminIDParsed() // Different from reporter
	imageID := ValidImageIDParsed()

	metadata, err := gallery.NewImageMetadata(
		"Test Image",
		"Description",
		"test-image.jpg",
		"image/jpeg",
		1920,
		1080,
		512000,
		"/storage/test-image.jpg",
		"local",
	)

  require.NoError(t, err)

	image := gallery.ReconstructImage(
		imageID,
		ownerID,
		metadata,
		gallery.VisibilityPublic,
		gallery.StatusActive,
		gallery.ScanStatusClean,
		[]gallery.ImageVariant{},
		[]gallery.Tag{},
		nil,
		0,
		0,
		0,
		time.Now(),
		time.Now(),
	)

	return image
}

// ValidImageOwnedByReporter creates a valid Image owned by the reporter (for testing self-report prevention).
func ValidImageOwnedByReporter(t *testing.T) *gallery.Image {
	t.Helper()

	ownerID := ValidUserIDParsed() // Same as reporter
	imageID := ValidImageIDParsed()

	metadata, err := gallery.NewImageMetadata(
		"Test Image",
		"Test description",
		"test-image.jpg",
		"image/jpeg",
		1920,
		1080,
		512000,
		"/storage/test-image.jpg",
		"local",
	)
  
	require.NoError(t, err)

	image := gallery.ReconstructImage(
		imageID,
		ownerID,
		metadata,
		gallery.VisibilityPublic,
		gallery.StatusActive,
		gallery.ScanStatusClean,
		[]gallery.ImageVariant{},
		[]gallery.Tag{},
		nil,
		0,
		0,
		0,
		time.Now(),
		time.Now(),
	)

	return image
}

// ValidReportInReviewing creates a report that's in reviewing status.
func ValidReportInReviewing(t *testing.T) *moderation.Report {
	t.Helper()

	reporterID := ValidUserIDParsed()
	imageID := ValidImageIDParsed()
	reason, _ := moderation.ParseReportReason(ValidReportReason)

	// Use ReconstructReport to create a report in reviewing status
	return moderation.ReconstructReport(
		ValidReportIDParsed(),
		reporterID,
		imageID,
		reason,
		ValidReportDesc,
		moderation.StatusReviewing,
		nil,        // resolvedBy
		nil,        // resolvedAt
		"",         // resolution
		time.Now(), // createdAt
	)
}

// ValidResolvedReport creates a report that's already resolved.
func ValidResolvedReport(t *testing.T) *moderation.Report {
	t.Helper()

	reporterID := ValidUserIDParsed()
	imageID := ValidImageIDParsed()
	adminID := ValidAdminIDParsed()
	reason, _ := moderation.ParseReportReason(ValidReportReason)
	now := time.Now()

	return moderation.ReconstructReport(
		ValidReportIDParsed(),
		reporterID,
		imageID,
		reason,
		ValidReportDesc,
		moderation.StatusResolved,
		&adminID,                           // resolvedBy
		&now,                               // resolvedAt
		"Report reviewed and action taken", // resolution
		time.Now().Add(-time.Hour),         // createdAt (1 hour ago)
	)
}

// ValidDismissedReport creates a report that's already dismissed.
func ValidDismissedReport(t *testing.T) *moderation.Report {
	t.Helper()

	reporterID := ValidUserIDParsed()
	imageID := ValidImageIDParsed()
	adminID := ValidAdminIDParsed()
	reason, _ := moderation.ParseReportReason(ValidReportReason)
	now := time.Now()

	return moderation.ReconstructReport(
		ValidReportIDParsed(),
		reporterID,
		imageID,
		reason,
		ValidReportDesc,
		moderation.StatusDismissed,
		&adminID,                   // resolvedBy
		&now,                       // resolvedAt
		"Report dismissed",         // resolution
		time.Now().Add(-time.Hour), // createdAt (1 hour ago)
	)
}
