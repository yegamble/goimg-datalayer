package handlers

import (
	"time"

	galleryqueries "github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	moderationqueries "github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

// HTTP-specific request DTOs for the handlers layer.
// These DTOs are separate from application-layer DTOs and represent the HTTP contract.
// They include JSON tags and validation rules using go-playground/validator.

// RegisterRequest represents the HTTP request body for user registration.
// POST /api/v1/auth/register.
type RegisterRequest struct {
	Email    string `json:"email"    validate:"required,email,max=255"`
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password string `json:"password" validate:"required,min=12,max=128"`
}

// LoginRequest represents the HTTP request body for user login.
// POST /api/v1/auth/login
//
// Identifier can be either email or username.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest represents the HTTP request body for token refresh.
// POST /api/v1/auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest represents the HTTP request body for logout.
// POST /api/v1/auth/logout
//
// Both fields are optional. If neither is provided, logout uses the session from JWT context.
// If logout_all is true, all sessions for the user are revoked.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
	LogoutAll    bool   `json:"logout_all,omitempty"`
}

// UpdateUserRequest represents the HTTP request body for updating user profile.
// PUT /api/v1/users/{id}
//
// All fields are optional (use pointers to indicate "no change").
// Only provided fields will be updated.
type UpdateUserRequest struct {
	DisplayName *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	Bio         *string `json:"bio,omitempty"          validate:"omitempty,max=500"`
}

// DeleteUserRequest represents the HTTP request body for deleting a user account.
// DELETE /api/v1/users/{id}
//
// Requires password confirmation to prevent accidental deletion.
type DeleteUserRequest struct {
	Password string `json:"password" validate:"required"`
}

// ============================================================================
// Gallery DTOs - Image Management
// ============================================================================

// UploadImageRequest represents the HTTP request for image upload.
// POST /api/v1/images
//
// Note: This is used with multipart/form-data. Fields are extracted from the form.
// Validation is performed at the application layer.
type UploadImageRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Visibility  string   `json:"visibility,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// UploadImageResponse represents the HTTP response after image upload.
type UploadImageResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// UpdateImageRequest represents the HTTP request body for updating image metadata.
// PUT /api/v1/images/{id}
//
// All fields are optional. Only provided fields will be updated.
type UpdateImageRequest struct {
	Title       *string  `json:"title,omitempty"       validate:"omitempty,max=255"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=2000"`
	Visibility  *string  `json:"visibility,omitempty"  validate:"omitempty,oneof=public private unlisted"`
	Tags        []string `json:"tags,omitempty"        validate:"omitempty,dive,max=50"`
}

// PaginatedImagesResponse represents a paginated list of images.
type PaginatedImagesResponse struct {
	Images     []ImageDTO `json:"images"`
	TotalCount int64      `json:"total_count"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
}

// ImageDTO is an alias for the application layer ImageDTO to avoid duplication.
type ImageDTO = galleryqueries.ImageDTO

// AlbumDTO is an alias for the application layer AlbumDTO to avoid duplication.
type AlbumDTO = galleryqueries.AlbumDTO

// VariantDTO is an alias for the application layer VariantDTO to avoid duplication.
type VariantDTO = galleryqueries.VariantDTO

// TagDTO is an alias for the application layer TagDTO to avoid duplication.
type TagDTO = galleryqueries.TagDTO

// ============================================================================
// Gallery DTOs - Album Management
// ============================================================================

// CreateAlbumRequest represents the HTTP request body for creating an album.
// POST /api/v1/albums.
type CreateAlbumRequest struct {
	Title       string `json:"title"                 validate:"required,max=255"`
	Description string `json:"description,omitempty" validate:"omitempty,max=2000"`
	Visibility  string `json:"visibility,omitempty"  validate:"omitempty,oneof=public private unlisted"`
}

// UpdateAlbumRequest represents the HTTP request body for updating album metadata.
// PUT /api/v1/albums/{id}
//
// All fields are optional. Only provided fields will be updated.
type UpdateAlbumRequest struct {
	Title        *string `json:"title,omitempty"          validate:"omitempty,max=255"`
	Description  *string `json:"description,omitempty"    validate:"omitempty,max=2000"`
	Visibility   *string `json:"visibility,omitempty"     validate:"omitempty,oneof=public private unlisted"`
	CoverImageID *string `json:"cover_image_id,omitempty" validate:"omitempty,uuid"`
	ParentID     *string `json:"parent_id,omitempty"      validate:"omitempty"` // Empty string removes parent
}

// AddImageToAlbumRequest represents the HTTP request body for adding an image to an album.
// POST /api/v1/albums/{albumID}/images.
type AddImageToAlbumRequest struct {
	ImageID string `json:"image_id" validate:"required,uuid"`
}

// PaginatedAlbumsResponse represents a paginated list of albums.
type PaginatedAlbumsResponse struct {
	Albums     []AlbumDTO `json:"albums"`
	TotalCount int64      `json:"total_count"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
}

// AlbumDTO is imported from application/gallery/queries package.
// It's already defined there as queries.AlbumDTO.

// ============================================================================
// Gallery DTOs - Social Interactions
// ============================================================================

// LikeResponse represents the HTTP response for like/unlike operations.
// POST /api/v1/images/{imageID}/like
// DELETE /api/v1/images/{imageID}/like.
type LikeResponse struct {
	Liked     bool  `json:"liked"`
	LikeCount int64 `json:"like_count"`
}

// AddCommentRequest represents the HTTP request body for adding a comment.
// POST /api/v1/images/{imageID}/comments.
type AddCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// CommentResponse represents the HTTP response after adding a comment.
type CommentResponse struct {
	ID        string `json:"id"`
	ImageID   string `json:"image_id"`
	UserID    string `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at,omitempty"`
}

// CommentDTO represents a comment in list responses.
type CommentDTO struct {
	ID        string `json:"id"`
	ImageID   string `json:"image_id"`
	UserID    string `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// PaginatedCommentsResponse represents a paginated list of comments.
type PaginatedCommentsResponse struct {
	Comments []CommentDTO `json:"comments"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PerPage  int          `json:"per_page"`
}

// ============================================================================
// Two-Factor Authentication DTOs
// ============================================================================

// Verify2FARequest represents the HTTP request body for verifying 2FA setup.
// POST /api/v1/auth/2fa/verify
type Verify2FARequest struct {
	Code string `json:"code" validate:"required,len=6,numeric"`
}

// Disable2FARequest represents the HTTP request body for disabling 2FA.
// POST /api/v1/auth/2fa/disable
type Disable2FARequest struct {
	Password string `json:"password" validate:"required"`
	Code     string `json:"code,omitempty" validate:"omitempty,len=6,numeric"`
}

// Login2FARequest represents the HTTP request body for completing 2FA during login.
// POST /api/v1/auth/2fa/login
type Login2FARequest struct {
	PendingToken  string `json:"pending_token" validate:"required"`
	Code          string `json:"code" validate:"required"`
	UseBackupCode bool   `json:"use_backup_code"`
}

// RegenerateBackupCodesRequest represents the HTTP request body for regenerating backup codes.
// POST /api/v1/auth/2fa/backup-codes/regenerate
type RegenerateBackupCodesRequest struct {
	Password string `json:"password" validate:"required"`
}

// Verify2FALoginRequest represents the HTTP request body for verifying 2FA during login (Sprint 11).
// POST /api/v1/auth/2fa/login-verify
//
// This endpoint is used when a user with 2FA enabled logs in and receives a non-elevated token.
// The user must provide their TOTP code to upgrade to an elevated session.
type Verify2FALoginRequest struct {
	Code          string `json:"code" validate:"required"`
	UseBackupCode bool   `json:"use_backup_code"`
}

// ============================================================================
// Moderation DTOs (Sprint 14)
// ============================================================================

// CreateReportRequest represents the HTTP request body for creating an abuse report.
// POST /api/v1/reports
type CreateReportRequest struct {
	ImageID     string `json:"image_id" validate:"required,uuid"`
	Reason      string `json:"reason" validate:"required,oneof=spam inappropriate copyright other"`
	Description string `json:"description" validate:"required,min=10,max=2000"`
}

// CreateReportResponse represents the HTTP response after creating a report.
type CreateReportResponse struct {
	ID        string  `json:"id"`
	Status    string  `json:"status"`
	CreatedAt *string `json:"created_at,omitempty"`
}

// ListPendingReportsResponse represents a paginated list of pending reports.
type ListPendingReportsResponse struct {
	Reports    []ReportDTO `json:"reports"`
	TotalCount int64       `json:"total_count"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
}

// ReportDTO is an alias for the application layer ReportDTO to avoid duplication.
type ReportDTO = moderationqueries.ReportDTO

// ResolveReportRequest represents the HTTP request body for resolving a report.
// POST /api/v1/moderation/reports/{reportID}/resolve
type ResolveReportRequest struct {
	Resolution string `json:"resolution" validate:"required,min=10,max=2000"`
}

// BanUserRequest represents the HTTP request body for banning a user.
// POST /api/v1/users/{userID}/ban
type BanUserRequest struct {
	Reason        string `json:"reason" validate:"required,min=10,max=500"`
	DurationHours *int   `json:"duration_hours,omitempty" validate:"omitempty,min=1,max=87600"` // Max 10 years
}

// BanUserResponse represents the HTTP response after banning a user.
type BanUserResponse struct {
	BanID     string     `json:"ban_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ListActiveBansResponse represents a list of all active bans.
type ListActiveBansResponse struct {
	Bans       []BanDTO `json:"bans"`
	TotalCount int64    `json:"total_count"`
}

// BanDTO is an alias for the application layer BanDTO to avoid duplication.
type BanDTO = moderationqueries.BanDTO

// ============================================================================
// NSFW Detection DTOs (Sprint 15)
// ============================================================================

// ScanImageNSFWRequest represents the HTTP request body for initiating an NSFW scan.
// POST /api/v1/moderation/nsfw/scan
type ScanImageNSFWRequest struct {
	ImageID string `json:"image_id" validate:"required,uuid"`
	Force   bool   `json:"force,omitempty"` // Force rescan even if active scan exists
}

// ScanImageNSFWResponse represents the HTTP response after initiating an NSFW scan.
type ScanImageNSFWResponse struct {
	ScanID         string  `json:"scan_id"`
	Status         string  `json:"status"`
	Category       string  `json:"category"`
	Score          float64 `json:"score"`
	IsNSFW         bool    `json:"is_nsfw"`
	RequiresReview bool    `json:"requires_review"`
	Provider       string  `json:"provider"`
}

// ListNSFWFlaggedResponse represents a paginated list of NSFW flagged images.
type ListNSFWFlaggedResponse struct {
	Scans      []*NSFWScanDTO `json:"scans"`
	TotalCount int64          `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int64          `json:"total_pages"`
}

// NSFWScanDTO is an alias for the application layer NSFWScanDTO to avoid duplication.
type NSFWScanDTO = moderationqueries.NSFWScanDTO

// ListNSFWScansByImageResponse represents all NSFW scans for an image.
type ListNSFWScansByImageResponse struct {
	Scans      []*NSFWScanDTO `json:"scans"`
	TotalCount int            `json:"total_count"`
}

// ============================================================================
// Groups/Communities DTOs (Sprint 20)
// ============================================================================

// CreateGroupRequest represents the HTTP request body for creating a group.
// POST /api/v1/groups
type CreateGroupRequest struct {
	Name        string                   `json:"name" validate:"required,min=3,max=100"`
	Slug        string                   `json:"slug" validate:"required,min=3,max=100"`
	Description string                   `json:"description,omitempty" validate:"omitempty,max=1000"`
	GroupType   string                   `json:"group_type" validate:"required,oneof=public private invite-only"`
	Settings    *community.GroupSettings `json:"settings,omitempty"`
}

// UpdateGroupRequest represents the HTTP request body for updating a group.
// PUT /api/v1/groups/{groupID}
type UpdateGroupRequest struct {
	Description *string                  `json:"description,omitempty" validate:"omitempty,max=1000"`
	Settings    *community.GroupSettings `json:"settings,omitempty"`
}

// UpdateMemberRoleRequest represents the HTTP request body for updating a member's role.
// PUT /api/v1/groups/{groupID}/members/{userID}/role
type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=owner admin moderator member"`
}

// BanMemberRequest represents the HTTP request body for banning a member.
// POST /api/v1/groups/{groupID}/members/{userID}/ban
type BanMemberRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=500"`
}

// GroupResponse represents a group in HTTP responses.
type GroupResponse struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Slug         string                `json:"slug"`
	Description  string                `json:"description"`
	GroupType    string                `json:"group_type"`
	OwnerID      string                `json:"owner_id"`
	Settings     GroupSettingsResponse `json:"settings"`
	MemberCount  int                   `json:"member_count"`
	ImageCount   int                   `json:"image_count"`
	AlbumCount   int                   `json:"album_count"`
	CoverImageID *string               `json:"cover_image_id,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// GroupSettingsResponse represents group settings in HTTP responses.
type GroupSettingsResponse struct {
	MaxMembers         int  `json:"max_members"`
	RequireApproval    bool `json:"require_approval"`
	AllowMemberInvites bool `json:"allow_member_invites"`
	AllowMemberAlbums  bool `json:"allow_member_albums"`
}

// MembershipResponse represents a group membership in HTTP responses.
type MembershipResponse struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"group_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	JoinedAt  time.Time `json:"joined_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PaginatedGroupsResponse represents a paginated list of groups.
type PaginatedGroupsResponse struct {
	Groups     []GroupResponse `json:"groups"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int64           `json:"total_pages"`
}

// SearchGroupsResponse represents search results for groups.
type SearchGroupsResponse struct {
	Groups     []GroupResponse `json:"groups"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int64           `json:"total_pages"`
	Query      string          `json:"query"`
}

// PaginatedMembersResponse represents a paginated list of group members.
type PaginatedMembersResponse struct {
	Members    []MembershipResponse `json:"members"`
	TotalCount int64                `json:"total_count"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
	TotalPages int64                `json:"total_pages"`
}

// PaginatedUserGroupsResponse represents a paginated list of groups for a user.
type PaginatedUserGroupsResponse struct {
	Memberships []MembershipResponse `json:"memberships"`
	TotalCount  int64                `json:"total_count"`
	Page        int                  `json:"page"`
	PerPage     int                  `json:"per_page"`
	TotalPages  int64                `json:"total_pages"`
}

// ============================================================================
// Group Invitations DTOs (Sprint 20 - S20-GROUP-006)
// ============================================================================

// InviteToGroupRequest represents the HTTP request body for inviting a user to a group.
// POST /api/v1/groups/{groupID}/invitations
//
// Either email OR user_id must be provided (mutually exclusive).
type InviteToGroupRequest struct {
	Email  *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	UserID *string `json:"user_id,omitempty" validate:"omitempty,uuid"`
}

// InvitationResponse represents a group invitation in HTTP responses.
type InvitationResponse struct {
	ID        string     `json:"id"`
	GroupID   string     `json:"group_id"`
	InvitedBy string     `json:"invited_by"`
	Email     *string    `json:"email,omitempty"`
	UserID    *string    `json:"user_id,omitempty"`
	Token     string     `json:"token"` // Secure token for accepting invitation
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// PaginatedInvitationsResponse represents a paginated list of group invitations.
type PaginatedInvitationsResponse struct {
	Invitations []InvitationResponse `json:"invitations"`
	TotalCount  int                  `json:"total_count"`
}

// ============================================================================
// Group Images DTOs (Sprint 20 - S20-GROUP-005 Moderation Queue)
// ============================================================================

// ShareImageRequest represents the HTTP request body for sharing an image to a group.
// POST /api/v1/groups/{groupID}/images
type ShareImageRequest struct {
	ImageID string `json:"image_id" validate:"required,uuid"`
}

// RejectImageRequest represents the HTTP request body for rejecting a group image.
// POST /api/v1/groups/{groupID}/images/{groupImageID}/reject
type RejectImageRequest struct {
	Reason string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// GroupImageResponse represents a group image in HTTP responses.
type GroupImageResponse struct {
	ID         string     `json:"id"`
	GroupID    string     `json:"group_id"`
	ImageID    string     `json:"image_id"`
	SharedBy   string     `json:"shared_by"`
	Status     string     `json:"status"` // pending, approved, rejected
	ReviewedBy *string    `json:"reviewed_by,omitempty"`
	SharedAt   *time.Time `json:"shared_at,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
}

// PaginatedGroupImagesResponse represents a paginated list of group images.
type PaginatedGroupImagesResponse struct {
	Images     []GroupImageResponse `json:"images"`
	TotalCount int64                `json:"total_count"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
	TotalPages int64                `json:"total_pages"`
}

// ============================================================================
// Group Albums DTOs (Sprint 20)
// ============================================================================

// CreateGroupAlbumRequest represents the HTTP request body for creating a group album.
// POST /api/v1/groups/{groupID}/albums
type CreateGroupAlbumRequest struct {
	Title       string `json:"title" validate:"required,max=255"`
	Description string `json:"description,omitempty" validate:"omitempty,max=2000"`
	IsPublic    bool   `json:"is_public"`
}

// UpdateGroupAlbumRequest represents the HTTP request body for updating a group album.
// PUT /api/v1/groups/{groupID}/albums/{albumID}
type UpdateGroupAlbumRequest struct {
	Title        *string `json:"title,omitempty" validate:"omitempty,max=255"`
	Description  *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	CoverImageID *string `json:"cover_image_id,omitempty" validate:"omitempty,uuid"`
}

// AddImageToGroupAlbumRequest represents the HTTP request body for adding an image to a group album.
// POST /api/v1/groups/{groupID}/albums/{albumID}/images
type AddImageToGroupAlbumRequest struct {
	ImageID string `json:"image_id" validate:"required,uuid"`
}

// GroupAlbumResponse represents a group album in HTTP responses.
type GroupAlbumResponse struct {
	ID           string    `json:"id"`
	GroupID      string    `json:"group_id"`
	CreatedBy    string    `json:"created_by"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	CoverImageID *string   `json:"cover_image_id,omitempty"`
	ImageCount   int       `json:"image_count"`
	IsPublic     bool      `json:"is_public"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PaginatedGroupAlbumsResponse represents a paginated list of group albums.
type PaginatedGroupAlbumsResponse struct {
	Albums     []GroupAlbumResponse `json:"albums"`
	TotalCount int64                `json:"total_count"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
	TotalPages int64                `json:"total_pages"`
}
