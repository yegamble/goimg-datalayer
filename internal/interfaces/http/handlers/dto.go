package handlers

import (
	"time"

	galleryqueries "github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	moderationqueries "github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

type RegisterRequest struct {
	Email    string `json:"email"    validate:"required,email,max=255"`
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password string `json:"password" validate:"required,min=12,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
	LogoutAll    bool   `json:"logout_all,omitempty"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"        validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=12,max=128"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

type UpdateUserRequest struct {
	DisplayName *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	Bio         *string `json:"bio,omitempty"          validate:"omitempty,max=500"`
}

type DeleteUserRequest struct {
	Password string `json:"password" validate:"required"`
}

// Note: This is used with multipart/form-data. Fields are extracted from the form.
type UploadImageRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Visibility  string   `json:"visibility,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type UploadImageResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type UpdateImageRequest struct {
	Title       *string  `json:"title,omitempty"       validate:"omitempty,max=255"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=2000"`
	Visibility  *string  `json:"visibility,omitempty"  validate:"omitempty,oneof=public private unlisted"`
	Tags        []string `json:"tags,omitempty"        validate:"omitempty,dive,max=50"`
}

type PaginatedImagesResponse struct {
	Images     []ImageDTO `json:"images"`
	TotalCount int64      `json:"total_count"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
}

type ImageDTO = galleryqueries.ImageDTO

type AlbumDTO = galleryqueries.AlbumDTO

type VariantDTO = galleryqueries.VariantDTO

type TagDTO = galleryqueries.TagDTO

type CreateAlbumRequest struct {
	Title       string `json:"title"                 validate:"required,max=255"`
	Description string `json:"description,omitempty" validate:"omitempty,max=2000"`
	Visibility  string `json:"visibility,omitempty"  validate:"omitempty,oneof=public private unlisted"`
}

type UpdateAlbumRequest struct {
	Title        *string `json:"title,omitempty"          validate:"omitempty,max=255"`
	Description  *string `json:"description,omitempty"    validate:"omitempty,max=2000"`
	Visibility   *string `json:"visibility,omitempty"     validate:"omitempty,oneof=public private unlisted"`
	CoverImageID *string `json:"cover_image_id,omitempty" validate:"omitempty,uuid"`
	ParentID     *string `json:"parent_id,omitempty"      validate:"omitempty"`
}

type AddImageToAlbumRequest struct {
	ImageID string `json:"image_id" validate:"required,uuid"`
}

type PaginatedAlbumsResponse struct {
	Albums     []AlbumDTO `json:"albums"`
	TotalCount int64      `json:"total_count"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
}

type LikeResponse struct {
	Liked     bool  `json:"liked"`
	LikeCount int64 `json:"like_count"`
}

type AddCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

type CommentResponse struct {
	ID        string `json:"id"`
	ImageID   string `json:"image_id"`
	UserID    string `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at,omitempty"`
}

type CommentDTO struct {
	ID        string `json:"id"`
	ImageID   string `json:"image_id"`
	UserID    string `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type PaginatedCommentsResponse struct {
	Comments []CommentDTO `json:"comments"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PerPage  int          `json:"per_page"`
}

type Verify2FARequest struct {
	Code string `json:"code" validate:"required,len=6,numeric"`
}

type Disable2FARequest struct {
	Password string `json:"password" validate:"required"`
	Code     string `json:"code,omitempty" validate:"omitempty,len=6,numeric"`
}

type Login2FARequest struct {
	PendingToken  string `json:"pending_token" validate:"required"`
	Code          string `json:"code" validate:"required"`
	UseBackupCode bool   `json:"use_backup_code"`
}

type RegenerateBackupCodesRequest struct {
	Password string `json:"password" validate:"required"`
}

type Verify2FALoginRequest struct {
	Code          string `json:"code" validate:"required"`
	UseBackupCode bool   `json:"use_backup_code"`
}

type CreateReportRequest struct {
	ImageID     string `json:"image_id" validate:"required,uuid"`
	Reason      string `json:"reason" validate:"required,oneof=spam inappropriate copyright other"`
	Description string `json:"description" validate:"required,min=10,max=2000"`
}

type CreateReportResponse struct {
	ID        string  `json:"id"`
	Status    string  `json:"status"`
	CreatedAt *string `json:"created_at,omitempty"`
}

type ListPendingReportsResponse struct {
	Reports    []ReportDTO `json:"reports"`
	TotalCount int64       `json:"total_count"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
}

type ReportDTO = moderationqueries.ReportDTO

type ResolveReportRequest struct {
	Resolution string `json:"resolution" validate:"required,min=10,max=2000"`
}

type BanUserRequest struct {
	Reason        string `json:"reason" validate:"required,min=10,max=500"`
	DurationHours *int   `json:"duration_hours,omitempty" validate:"omitempty,min=1,max=87600"`
}

type BanUserResponse struct {
	BanID     string     `json:"ban_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type ListActiveBansResponse struct {
	Bans       []BanDTO `json:"bans"`
	TotalCount int64    `json:"total_count"`
}

type BanDTO = moderationqueries.BanDTO

type ScanImageNSFWRequest struct {
	ImageID string `json:"image_id" validate:"required,uuid"`
	Force   bool   `json:"force,omitempty"`
}

type ScanImageNSFWResponse struct {
	ScanID         string  `json:"scan_id"`
	Status         string  `json:"status"`
	Category       string  `json:"category"`
	Score          float64 `json:"score"`
	IsNSFW         bool    `json:"is_nsfw"`
	RequiresReview bool    `json:"requires_review"`
	Provider       string  `json:"provider"`
}

type ListNSFWFlaggedResponse struct {
	Scans      []*NSFWScanDTO `json:"scans"`
	TotalCount int64          `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int64          `json:"total_pages"`
}

type NSFWScanDTO = moderationqueries.NSFWScanDTO

type ListNSFWScansByImageResponse struct {
	Scans      []*NSFWScanDTO `json:"scans"`
	TotalCount int            `json:"total_count"`
}

type CreateGroupRequest struct {
	Name        string                   `json:"name" validate:"required,min=3,max=100"`
	Slug        string                   `json:"slug" validate:"required,min=3,max=100"`
	Description string                   `json:"description,omitempty" validate:"omitempty,max=1000"`
	GroupType   string                   `json:"group_type" validate:"required,oneof=public private invite-only"`
	Settings    *community.GroupSettings `json:"settings,omitempty"`
}

type UpdateGroupRequest struct {
	Description *string                  `json:"description,omitempty" validate:"omitempty,max=1000"`
	Settings    *community.GroupSettings `json:"settings,omitempty"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=owner admin moderator member"`
}

type BanMemberRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=500"`
}

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

type GroupSettingsResponse struct {
	MaxMembers         int  `json:"max_members"`
	RequireApproval    bool `json:"require_approval"`
	AllowMemberInvites bool `json:"allow_member_invites"`
	AllowMemberAlbums  bool `json:"allow_member_albums"`
}

type MembershipResponse struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"group_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	JoinedAt  time.Time `json:"joined_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PaginatedGroupsResponse struct {
	Groups     []GroupResponse `json:"groups"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int64           `json:"total_pages"`
}

type SearchGroupsResponse struct {
	Groups     []GroupResponse `json:"groups"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int64           `json:"total_pages"`
	Query      string          `json:"query"`
}

type PaginatedMembersResponse struct {
	Members    []MembershipResponse `json:"members"`
	TotalCount int64                `json:"total_count"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
	TotalPages int64                `json:"total_pages"`
}

type PaginatedUserGroupsResponse struct {
	Memberships []MembershipResponse `json:"memberships"`
	TotalCount  int64                `json:"total_count"`
	Page        int                  `json:"page"`
	PerPage     int                  `json:"per_page"`
	TotalPages  int64                `json:"total_pages"`
}

type InviteToGroupRequest struct {
	Email  *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	UserID *string `json:"user_id,omitempty" validate:"omitempty,uuid"`
}

type InvitationResponse struct {
	ID        string     `json:"id"`
	GroupID   string     `json:"group_id"`
	InvitedBy string     `json:"invited_by"`
	Email     *string    `json:"email,omitempty"`
	UserID    *string    `json:"user_id,omitempty"`
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type PaginatedInvitationsResponse struct {
	Invitations []InvitationResponse `json:"invitations"`
	TotalCount  int                  `json:"total_count"`
}

type ShareImageRequest struct {
	ImageID string `json:"image_id" validate:"required,uuid"`
}

type RejectImageRequest struct {
	Reason string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

type GroupImageResponse struct {
	ID         string     `json:"id"`
	GroupID    string     `json:"group_id"`
	ImageID    string     `json:"image_id"`
	SharedBy   string     `json:"shared_by"`
	Status     string     `json:"status"`
	ReviewedBy *string    `json:"reviewed_by,omitempty"`
	SharedAt   *time.Time `json:"shared_at,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
}

type PaginatedGroupImagesResponse struct {
	Images     []GroupImageResponse `json:"images"`
	TotalCount int64                `json:"total_count"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
	TotalPages int64                `json:"total_pages"`
}

type CreateGroupAlbumRequest struct {
	Title       string `json:"title" validate:"required,max=255"`
	Description string `json:"description,omitempty" validate:"omitempty,max=2000"`
	IsPublic    bool   `json:"is_public"`
}

type UpdateGroupAlbumRequest struct {
	Title        *string `json:"title,omitempty" validate:"omitempty,max=255"`
	Description  *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	CoverImageID *string `json:"cover_image_id,omitempty" validate:"omitempty,uuid"`
}

type AddImageToGroupAlbumRequest struct {
	ImageID string `json:"image_id" validate:"required,uuid"`
}

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

type PaginatedGroupAlbumsResponse struct {
	Albums     []GroupAlbumResponse `json:"albums"`
	TotalCount int64                `json:"total_count"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
	TotalPages int64                `json:"total_pages"`
}
