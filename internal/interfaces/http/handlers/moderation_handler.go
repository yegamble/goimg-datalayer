package handlers

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// ModerationHandler handles moderation-related HTTP endpoints.
// It delegates to application layer command and query handlers for business logic.
type ModerationHandler struct {
	// Report command handlers
	createReport  *commands.CreateReportHandler
	startReview   *commands.StartReviewHandler
	resolveReport *commands.ResolveReportHandler
	dismissReport *commands.DismissReportHandler

	// Ban command handlers
	banUser   *commands.BanUserHandler
	unbanUser *commands.UnbanUserHandler

	// NSFW command handlers
	scanNSFW *commands.ScanImageNSFWHandler

	// Report query handlers
	getReport   *queries.GetReportHandler
	listPending *queries.ListPendingReportsHandler

	// Ban query handlers
	getBanStatus *queries.GetUserBanStatusHandler
	listBans     *queries.ListActiveBansHandler

	// NSFW query handlers
	getNSFWScan          *queries.GetNSFWScanHandler
	listNSFWFlagged      *queries.ListNSFWFlaggedHandler
	listNSFWScansByImage *queries.ListNSFWScansByImageHandler

	logger zerolog.Logger
}

// NewModerationHandler creates a new ModerationHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewModerationHandler(
	createReport *commands.CreateReportHandler,
	startReview *commands.StartReviewHandler,
	resolveReport *commands.ResolveReportHandler,
	dismissReport *commands.DismissReportHandler,
	banUser *commands.BanUserHandler,
	unbanUser *commands.UnbanUserHandler,
	scanNSFW *commands.ScanImageNSFWHandler,
	getReport *queries.GetReportHandler,
	listPending *queries.ListPendingReportsHandler,
	getBanStatus *queries.GetUserBanStatusHandler,
	listBans *queries.ListActiveBansHandler,
	getNSFWScan *queries.GetNSFWScanHandler,
	listNSFWFlagged *queries.ListNSFWFlaggedHandler,
	listNSFWScansByImage *queries.ListNSFWScansByImageHandler,
	logger zerolog.Logger,
) *ModerationHandler {
	return &ModerationHandler{
		createReport:         createReport,
		startReview:          startReview,
		resolveReport:        resolveReport,
		dismissReport:        dismissReport,
		banUser:              banUser,
		unbanUser:            unbanUser,
		scanNSFW:             scanNSFW,
		getReport:            getReport,
		listPending:          listPending,
		getBanStatus:         getBanStatus,
		listBans:             listBans,
		getNSFWScan:          getNSFWScan,
		listNSFWFlagged:      listNSFWFlagged,
		listNSFWScansByImage: listNSFWScansByImage,
		logger:               logger,
	}
}

// ============================================================================
// User Endpoints (Authenticated Users)
// ============================================================================

// CreateReport handles POST /api/v1/reports
// Creates a new abuse report for an image.
//
// Request body:
//   - image_id (string, required): UUID of the image being reported
//   - reason (string, required): Report reason (spam, inappropriate, copyright, other)
//   - description (string, required): Detailed description of the issue
//
// Response: 201 Created with CreateReportResponse
// Errors:
//   - 400: Invalid request data or validation failure
//   - 401: Not authenticated
//   - 404: Image not found
//   - 409: Cannot report your own content
//   - 500: Internal server error
//
// Rate limit: 10/hour (TODO: implement in middleware)
func (h *ModerationHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context (authenticated user)
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in create report handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Decode and validate request body
	var req CreateReportRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid create report request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid report data",
			validationErrors,
		)
		return
	}

	// 3. Build command
	cmd := commands.CreateReportCommand{
		ReporterID:  userCtx.UserID.String(),
		ImageID:     req.ImageID,
		Reason:      req.Reason,
		Description: req.Description,
	}

	// 4. Execute command
	result, err := h.createReport.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "create report")
		return
	}

	// 5. Return response
	h.logger.Info().
		Str("report_id", result.ReportID).
		Str("reporter_id", userCtx.UserID.String()).
		Str("image_id", req.ImageID).
		Msg("report created successfully")

	response := CreateReportResponse{
		ID:        result.ReportID,
		Status:    result.Status,
		CreatedAt: nil, // TODO: Add CreatedAt to command result if needed
	}

	if err := EncodeJSON(w, http.StatusCreated, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode create report response")
	}
}

// ============================================================================
// Admin/Moderator Endpoints
// ============================================================================

// ListPendingReports handles GET /api/v1/moderation/reports
// Lists pending reports for moderator queue.
//
// Query parameters:
//   - page (int, default: 1): Page number (1-indexed)
//   - per_page (int, default: 20, max: 100): Items per page
//
// Response: 200 OK with ListPendingReportsResponse
// Errors:
//   - 400: Invalid query parameters
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator or admin)
//   - 500: Internal server error
//
// Auth: Moderator or admin role required
func (h *ModerationHandler) ListPendingReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Parse query parameters
	page, err := parseIntParam(r.URL.Query().Get("page"), 1)
	if err != nil || page < 1 {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid page parameter",
		)
		return
	}

	perPage, err := parseIntParam(r.URL.Query().Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid per_page parameter",
		)
		return
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	// 2. Build query
	query := queries.ListPendingReportsQuery{
		Page:    page,
		PerPage: perPage,
	}

	// 3. Execute query
	result, err := h.listPending.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list pending reports")
		return
	}

	// 4. Return response
	h.logger.Debug().
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Reports)).
		Int64("total_count", result.TotalCount).
		Msg("pending reports listed successfully")

	response := ListPendingReportsResponse{
		Reports:    result.Reports,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PerPage:    result.PerPage,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list pending reports response")
	}
}

// GetReport handles GET /api/v1/moderation/reports/{reportID}
// Retrieves a single report by its ID.
//
// Path parameters:
//   - reportID: UUID of the report
//
// Response: 200 OK with ReportDTO
// Errors:
//   - 400: Invalid report ID format
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator or admin)
//   - 404: Report not found
//   - 500: Internal server error
//
// Auth: Moderator or admin role required
func (h *ModerationHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract report ID from path
	reportID := GetPathParam(r, "reportID")
	if reportID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing report ID",
		)
		return
	}

	// 2. Build query
	query := queries.GetReportQuery{
		ReportID: reportID,
	}

	// 3. Execute query
	report, err := h.getReport.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get report")
		return
	}

	// 4. Return report
	h.logger.Debug().
		Str("report_id", reportID).
		Msg("report retrieved successfully")

	if err := EncodeJSON(w, http.StatusOK, report); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get report response")
	}
}

// StartReview handles POST /api/v1/moderation/reports/{reportID}/review
// Starts a review for a pending report (transitions to under_review status).
//
// Path parameters:
//   - reportID: UUID of the report
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid report ID format
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator or admin)
//   - 404: Report not found
//   - 409: Report not in pending status
//   - 500: Internal server error
//
// Auth: Moderator or admin role required
func (h *ModerationHandler) StartReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in start review handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract report ID from path
	reportID := GetPathParam(r, "reportID")
	if reportID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing report ID",
		)
		return
	}

	// 3. Build command
	cmd := commands.StartReviewCommand{
		ReportID: reportID,
	}

	// 4. Execute command
	_, err = h.startReview.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "start review")
		return
	}

	// 5. Return 204 No Content
	h.logger.Info().
		Str("report_id", reportID).
		Str("reviewer_id", userCtx.UserID.String()).
		Msg("review started successfully")

	w.WriteHeader(http.StatusNoContent)
}

// ResolveReport handles POST /api/v1/moderation/reports/{reportID}/resolve
// Resolves a report with moderator action.
//
// Path parameters:
//   - reportID: UUID of the report
//
// Request body:
//   - resolution (string, required): Resolution notes explaining the action taken
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid request data or report ID
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator or admin)
//   - 404: Report not found
//   - 409: Report already resolved or in invalid state
//   - 500: Internal server error
//
// Auth: Moderator or admin role required
func (h *ModerationHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in resolve report handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract report ID from path
	reportID := GetPathParam(r, "reportID")
	if reportID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing report ID",
		)
		return
	}

	// 3. Decode and validate request body
	var req ResolveReportRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid resolve report request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid resolution data",
			validationErrors,
		)
		return
	}

	// 4. Build command
	cmd := commands.ResolveReportCommand{
		ReportID:   reportID,
		ResolverID: userCtx.UserID.String(),
		Resolution: req.Resolution,
	}

	// 5. Execute command
	_, err = h.resolveReport.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "resolve report")
		return
	}

	// 6. Return 204 No Content
	h.logger.Info().
		Str("report_id", reportID).
		Str("resolver_id", userCtx.UserID.String()).
		Msg("report resolved successfully")

	w.WriteHeader(http.StatusNoContent)
}

// DismissReport handles POST /api/v1/moderation/reports/{reportID}/dismiss
// Dismisses a report without action.
//
// Path parameters:
//   - reportID: UUID of the report
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid report ID format
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator or admin)
//   - 404: Report not found
//   - 409: Report already dismissed or in invalid state
//   - 500: Internal server error
//
// Auth: Moderator or admin role required
func (h *ModerationHandler) DismissReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in dismiss report handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract report ID from path
	reportID := GetPathParam(r, "reportID")
	if reportID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing report ID",
		)
		return
	}

	// 3. Build command
	cmd := commands.DismissReportCommand{
		ReportID:   reportID,
		ResolverID: userCtx.UserID.String(),
	}

	// 4. Execute command
	_, err = h.dismissReport.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "dismiss report")
		return
	}

	// 5. Return 204 No Content
	h.logger.Info().
		Str("report_id", reportID).
		Str("reviewer_id", userCtx.UserID.String()).
		Msg("report dismissed successfully")

	w.WriteHeader(http.StatusNoContent)
}

// BanUser handles POST /api/v1/users/{userID}/ban
// Bans a user from the platform.
//
// Path parameters:
//   - userID: UUID of the user to ban
//
// Request body:
//   - reason (string, required): Reason for the ban
//   - duration_hours (int, optional): Ban duration in hours (omit for permanent ban)
//
// Response: 201 Created with BanUserResponse
// Errors:
//   - 400: Invalid request data or user ID
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires admin role)
//   - 404: User not found
//   - 409: User already banned
//   - 500: Internal server error
//
// Auth: Admin role required
func (h *ModerationHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context (admin only)
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in ban user handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract user ID from path
	userID := GetPathParam(r, "userID")
	if userID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	// 3. Decode and validate request body
	var req BanUserRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid ban user request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid ban data",
			validationErrors,
		)
		return
	}

	// 4. Build command
	cmd := commands.BanUserCommand{
		UserID:        userID,
		BannedBy:      userCtx.UserID.String(),
		Reason:        req.Reason,
		DurationHours: req.DurationHours,
	}

	// 5. Execute command
	result, err := h.banUser.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "ban user")
		return
	}

	// 6. Return response
	h.logger.Info().
		Str("ban_id", result.BanID).
		Str("user_id", userID).
		Str("banned_by", userCtx.UserID.String()).
		Bool("is_permanent", result.IsPermanent).
		Msg("user banned successfully")

	response := BanUserResponse{
		BanID:     result.BanID,
		ExpiresAt: result.ExpiresAt,
	}

	if err := EncodeJSON(w, http.StatusCreated, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode ban user response")
	}
}

// UnbanUser handles DELETE /api/v1/users/{userID}/ban
// Removes a ban from a user.
//
// Path parameters:
//   - userID: UUID of the user to unban
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid user ID format
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires admin role)
//   - 404: User not found or not banned
//   - 500: Internal server error
//
// Auth: Admin role required
func (h *ModerationHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context (admin only)
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in unban user handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract user ID from path
	userID := GetPathParam(r, "userID")
	if userID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	// 3. Build command
	cmd := commands.UnbanUserCommand{
		UserID:    userID,
		RevokedBy: userCtx.UserID.String(),
	}

	// 4. Execute command
	_, err = h.unbanUser.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "unban user")
		return
	}

	// 5. Return 204 No Content
	h.logger.Info().
		Str("user_id", userID).
		Str("revoked_by", userCtx.UserID.String()).
		Msg("user unbanned successfully")

	w.WriteHeader(http.StatusNoContent)
}

// GetUserBanStatus handles GET /api/v1/users/{userID}/ban
// Checks if a user is currently banned.
//
// Path parameters:
//   - userID: UUID of the user
//
// Response: 200 OK with UserBanStatusResult
// Errors:
//   - 400: Invalid user ID format
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator/admin or viewing own status)
//   - 404: User not found
//   - 500: Internal server error
//
// Auth: Moderator, admin, or self
func (h *ModerationHandler) GetUserBanStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user ID from path
	userID := GetPathParam(r, "userID")
	if userID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	// 2. Build query
	query := queries.GetUserBanStatusQuery{
		UserID: userID,
	}

	// 3. Execute query
	result, err := h.getBanStatus.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get user ban status")
		return
	}

	// 4. Return result
	h.logger.Debug().
		Str("user_id", userID).
		Bool("is_banned", result.IsBanned).
		Msg("user ban status retrieved successfully")

	if err := EncodeJSON(w, http.StatusOK, result); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode ban status response")
	}
}

// ListActiveBans handles GET /api/v1/moderation/bans
// Lists all currently active bans.
//
// Query parameters:
//   - page (int, default: 1): Page number (1-indexed)
//   - per_page (int, default: 20, max: 100): Items per page
//
// Response: 200 OK with ListActiveBansResponse
// Errors:
//   - 400: Invalid query parameters
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires admin role)
//   - 500: Internal server error
//
// Auth: Admin role required
func (h *ModerationHandler) ListActiveBans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Build query (no pagination yet per the query handler)
	query := queries.ListActiveBansQuery{}

	// 2. Execute query
	result, err := h.listBans.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list active bans")
		return
	}

	// 3. Return response
	h.logger.Debug().
		Int("total_count", result.TotalCount).
		Msg("active bans listed successfully")

	response := ListActiveBansResponse{
		Bans:       result.Bans,
		TotalCount: int64(result.TotalCount),
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list active bans response")
	}
}

// ============================================================================
// NSFW Detection Endpoints (Sprint 15)
// ============================================================================

// ScanImageNSFW handles POST /api/v1/moderation/nsfw/scan
// Initiates an NSFW scan for an image.
//
// Request body:
//   - image_id (string, required): UUID of the image to scan
//   - force (bool, optional): Force rescan even if active scan exists
//
// Response: 201 Created with ScanImageNSFWResponse
// Errors:
//   - 400: Invalid request data or validation failure
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator or admin)
//   - 404: Image not found
//   - 409: Active scan already exists (unless force=true)
//   - 500: Internal server error
//   - 503: NSFW detection service unavailable
//
// Auth: Moderator or admin role required
func (h *ModerationHandler) ScanImageNSFW(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Decode and validate request body
	var req ScanImageNSFWRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid NSFW scan request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid NSFW scan request data",
			validationErrors,
		)
		return
	}

	// 2. Build command
	cmd := commands.ScanImageNSFWCommand{
		ImageID: req.ImageID,
		Force:   req.Force,
	}

	// 3. Execute command
	result, err := h.scanNSFW.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "scan image NSFW")
		return
	}

	// 4. Return response
	h.logger.Info().
		Str("scan_id", result.ScanID).
		Str("image_id", req.ImageID).
		Str("status", result.Status).
		Bool("is_nsfw", result.IsNSFW).
		Msg("NSFW scan completed")

	response := ScanImageNSFWResponse{
		ScanID:         result.ScanID,
		Status:         result.Status,
		Category:       result.Category,
		Score:          result.Score,
		IsNSFW:         result.IsNSFW,
		RequiresReview: result.RequiresReview,
		Provider:       result.Provider,
	}

	if err := EncodeJSON(w, http.StatusCreated, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode NSFW scan response")
	}
}

// GetNSFWScan handles GET /api/v1/moderation/nsfw/scans/{scanID}
// Retrieves a single NSFW scan by its ID.
//
// Path parameters:
//   - scanID: UUID of the scan
//
// Response: 200 OK with NSFWScanDTO
// Errors:
//   - 400: Invalid scan ID format
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator or admin)
//   - 404: Scan not found
//   - 500: Internal server error
//
// Auth: Moderator or admin role required
func (h *ModerationHandler) GetNSFWScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract scan ID from path
	scanID := GetPathParam(r, "scanID")
	if scanID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing scan ID",
		)
		return
	}

	// 2. Build query
	query := queries.GetNSFWScanQuery{
		ScanID: scanID,
	}

	// 3. Execute query
	scan, err := h.getNSFWScan.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get NSFW scan")
		return
	}

	// 4. Return scan
	h.logger.Debug().
		Str("scan_id", scanID).
		Msg("NSFW scan retrieved successfully")

	if err := EncodeJSON(w, http.StatusOK, scan); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get NSFW scan response")
	}
}

// ListNSFWFlagged handles GET /api/v1/moderation/nsfw/flagged
// Lists images flagged as NSFW for moderator review.
//
// Query parameters:
//   - page (int, default: 1): Page number (1-indexed)
//   - page_size (int, default: 20, max: 100): Items per page
//   - requires_review (bool, optional): Filter to only show items requiring review
//
// Response: 200 OK with ListNSFWFlaggedResponse
// Errors:
//   - 400: Invalid query parameters
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator or admin)
//   - 500: Internal server error
//
// Auth: Moderator or admin role required
func (h *ModerationHandler) ListNSFWFlagged(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Parse query parameters
	page, err := parseIntParam(r.URL.Query().Get("page"), 1)
	if err != nil || page < 1 {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid page parameter",
		)
		return
	}

	pageSize, err := parseIntParam(r.URL.Query().Get("page_size"), defaultPerPage)
	if err != nil || pageSize < 1 {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid page_size parameter",
		)
		return
	}
	if pageSize > maxPerPage {
		pageSize = maxPerPage
	}

	// Parse optional requires_review filter
	var requiresReview *bool
	if reqReviewStr := r.URL.Query().Get("requires_review"); reqReviewStr != "" {
		reqReviewBool := reqReviewStr == "true"
		requiresReview = &reqReviewBool
	}

	// 2. Build query
	query := queries.ListNSFWFlaggedQuery{
		Page:           page,
		PageSize:       pageSize,
		RequiresReview: requiresReview,
	}

	// 3. Execute query
	result, err := h.listNSFWFlagged.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list NSFW flagged")
		return
	}

	// 4. Return response
	h.logger.Debug().
		Int("page", page).
		Int("page_size", pageSize).
		Int("results", len(result.Scans)).
		Int64("total_count", result.TotalCount).
		Msg("NSFW flagged images listed successfully")

	response := ListNSFWFlaggedResponse{
		Scans:      result.Scans,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list NSFW flagged response")
	}
}

// ListNSFWScansByImage handles GET /api/v1/images/{imageID}/nsfw-scans
// Lists all NSFW scans for a specific image.
//
// Path parameters:
//   - imageID: UUID of the image
//
// Response: 200 OK with ListNSFWScansByImageResponse
// Errors:
//   - 400: Invalid image ID format
//   - 401: Not authenticated
//   - 403: Insufficient permissions (requires moderator, admin, or image owner)
//   - 404: Image not found
//   - 500: Internal server error
//
// Auth: Moderator, admin, or image owner
func (h *ModerationHandler) ListNSFWScansByImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 2. Build query
	query := queries.ListNSFWScansByImageQuery{
		ImageID: imageID,
	}

	// 3. Execute query
	result, err := h.listNSFWScansByImage.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list NSFW scans by image")
		return
	}

	// 4. Return response
	h.logger.Debug().
		Str("image_id", imageID).
		Int("total_count", result.TotalCount).
		Msg("NSFW scans by image listed successfully")

	response := ListNSFWScansByImageResponse{
		Scans:      result.Scans,
		TotalCount: result.TotalCount,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list NSFW scans by image response")
	}
}

// ============================================================================
// Error Mapping
// ============================================================================

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
func (h *ModerationHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("moderation operation failed")

	// Map domain errors to HTTP status codes
	switch {
	// Report errors
	case err.Error() == "report not found" || err.Error() == "find report by id: report not found":
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Report not found",
		)

	case err.Error() == "report already resolved" || err.Error() == "report not in pending status":
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"Report is not in a valid state for this operation",
		)

	case err.Error() == "cannot report your own content":
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"You cannot report your own content",
		)

	// Image errors
	case err.Error() == "image not found" || err.Error() == "find image: image not found":
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Image not found",
		)

	// User/Ban errors
	case err.Error() == "user not found" || err.Error() == "find user: user not found":
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"User not found",
		)

	case err.Error() == "user not banned" || err.Error() == "no active ban found":
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"User is not banned",
		)

	case err.Error() == "user already banned":
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"User is already banned",
		)

	// Validation errors
	case err.Error() == "invalid report id" || err.Error() == "invalid user id" || err.Error() == "invalid image id":
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid ID format",
		)

	case err.Error() == "invalid report reason":
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid report reason. Must be one of: spam, inappropriate, copyright, other",
		)

	// NSFW scan errors
	case containsError(err, "nsfw scan not found") || containsError(err, "find scan by id"):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"NSFW scan not found",
		)

	case containsError(err, "invalid scan id"):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid scan ID format",
		)

	case containsError(err, "nsfw detection disabled") || containsError(err, "no providers available"):
		middleware.WriteError(w, r,
			http.StatusServiceUnavailable,
			"Service Unavailable",
			"NSFW detection service is currently unavailable",
		)

	case containsError(err, "active scan exists"):
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"An active NSFW scan already exists for this image",
		)

	case containsError(err, "all providers failed"):
		middleware.WriteError(w, r,
			http.StatusServiceUnavailable,
			"Service Unavailable",
			"All NSFW detection providers failed",
		)

	// Default to internal error
	default:
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred",
		)
	}
}

// containsError checks if an error message contains a specific substring.
// This is used for flexible error matching as errors may be wrapped.
func containsError(err error, substr string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), substr)
}
