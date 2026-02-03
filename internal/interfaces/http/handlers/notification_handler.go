package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/notification/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/notification/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// NotificationHandler handles notification HTTP endpoints.
// It delegates to application layer command/query handlers for business logic.
type NotificationHandler struct {
	getNotificationsHandler *queries.GetNotificationsHandler
	getUnreadCountHandler   *queries.GetUnreadCountHandler
	markAsReadHandler       *commands.MarkNotificationsReadHandler
	logger                  zerolog.Logger
}

// NewNotificationHandler creates a new NotificationHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewNotificationHandler(
	getNotificationsHandler *queries.GetNotificationsHandler,
	getUnreadCountHandler *queries.GetUnreadCountHandler,
	markAsReadHandler *commands.MarkNotificationsReadHandler,
	logger zerolog.Logger,
) *NotificationHandler {
	return &NotificationHandler{
		getNotificationsHandler: getNotificationsHandler,
		getUnreadCountHandler:   getUnreadCountHandler,
		markAsReadHandler:       markAsReadHandler,
		logger:                  logger,
	}
}

// NotificationDTO represents a notification in HTTP responses.
type NotificationDTO struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Metadata  json.RawMessage `json:"metadata"`
	IsRead    bool            `json:"isRead"`
	CreatedAt time.Time       `json:"createdAt"`
	ReadAt    *time.Time      `json:"readAt,omitempty"`
}

// NotificationsListDTO represents a list of notifications with metadata.
type NotificationsListDTO struct {
	Notifications []NotificationDTO `json:"notifications"`
	TotalUnread   int               `json:"totalUnread"`
	Limit         int               `json:"limit"`
	Offset        int               `json:"offset"`
}

// UnreadCountDTO represents the count of unread notifications.
type UnreadCountDTO struct {
	Count int `json:"count"`
}

// MarkAsReadRequest represents the request body for marking notifications as read.
type MarkAsReadRequest struct {
	NotificationIDs []string `json:"notificationIds"` // Specific IDs to mark (optional)
	MarkAllAsRead   bool     `json:"markAllAsRead"`   // If true, mark all as read
}

// GetNotifications handles GET /api/v1/notifications
// Returns paginated list of notifications for the authenticated user.
//
// Query Parameters:
//   - unread_only: If true, only return unread notifications (default: false)
//   - limit: Maximum number of notifications to return (default: 20, max: 100)
//   - offset: Number of notifications to skip for pagination (default: 0)
//
// Response: 200 OK with NotificationsListDTO
// Errors:
//   - 400: Invalid query parameters
//   - 401: Not authenticated
//   - 500: Internal server error
func (h *NotificationHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract authenticated user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in notification handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Parse query parameters
	query := r.URL.Query()
	unreadOnly := query.Get("unread_only") == "true"
	limit := parseIntQueryParam(query.Get("limit"), 20, 1, 100)
	offset := parseIntQueryParam(query.Get("offset"), 0, 0, 1000000)

	// 3. Delegate to query handler
	result, err := h.getNotificationsHandler.Handle(ctx, queries.GetNotificationsQuery{
		UserID:     userCtx.UserID.String(),
		UnreadOnly: unreadOnly,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get notifications")
		return
	}

	// 4. Map to DTOs
	notificationDTOs := make([]NotificationDTO, 0, len(result.Notifications))
	for _, n := range result.Notifications {
		notificationDTOs = append(notificationDTOs, NotificationDTO{
			ID:        n.ID().String(),
			Type:      n.Type().String(),
			Title:     n.Title(),
			Body:      n.Body(),
			Metadata:  n.MetadataRaw(),
			IsRead:    n.IsRead(),
			CreatedAt: n.CreatedAt(),
			ReadAt:    n.ReadAt(),
		})
	}

	response := NotificationsListDTO{
		Notifications: notificationDTOs,
		TotalUnread:   result.TotalUnread,
		Limit:         limit,
		Offset:        offset,
	}

	// 5. Return response
	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode notifications response")
	}
}

// GetUnreadCount handles GET /api/v1/notifications/count
// Returns the count of unread notifications for the authenticated user.
//
// Response: 200 OK with UnreadCountDTO
// Errors:
//   - 401: Not authenticated
//   - 500: Internal server error
func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract authenticated user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in notification handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Delegate to query handler
	result, err := h.getUnreadCountHandler.Handle(ctx, queries.GetUnreadCountQuery{
		UserID: userCtx.UserID.String(),
	})
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get unread count")
		return
	}

	// 3. Return response
	response := UnreadCountDTO{
		Count: result.Count,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode unread count response")
	}
}

// MarkAsRead handles POST /api/v1/notifications/read
// Marks notifications as read for the authenticated user.
//
// Request Body: MarkAsReadRequest
//   - notificationIds: Array of notification IDs to mark as read (optional)
//   - markAllAsRead: If true, mark all notifications as read (optional)
//
// Response: 204 No Content on success
// Errors:
//   - 400: Invalid request body or parameters
//   - 401: Not authenticated
//   - 403: Notification does not belong to user
//   - 404: Notification not found
//   - 500: Internal server error
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract authenticated user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in notification handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Parse request body
	var req MarkAsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid request body in mark as read")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid request body",
		)
		return
	}

	// 3. Validate request
	if !req.MarkAllAsRead && len(req.NotificationIDs) == 0 {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Either notificationIds or markAllAsRead must be provided",
		)
		return
	}

	// 4. Delegate to command handler
	cmd := commands.MarkNotificationsReadCommand{
		UserID:          userCtx.UserID.String(),
		NotificationIDs: req.NotificationIDs,
		MarkAllAsRead:   req.MarkAllAsRead,
	}

	err = h.markAsReadHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "mark notifications as read")
		return
	}

	// 5. Return success
	w.WriteHeader(http.StatusNoContent)
}

// parseIntQueryParam parses an integer query parameter with default and bounds.
func parseIntQueryParam(value string, defaultVal, min, max int) int {
	if value == "" {
		return defaultVal
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultVal
	}

	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}

	return parsed
}

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
func (h *NotificationHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("notification operation failed")

	// Map specific domain errors to HTTP status codes
	switch {
	case errors.Is(err, notification.ErrNotificationNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Notification not found",
		)

	case errors.Is(err, notification.ErrRecipientRequired):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Recipient is required",
		)

	case errors.Is(err, notification.ErrTitleRequired):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Title is required",
		)

	case errors.Is(err, notification.ErrInvalidNotificationType):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid notification type",
		)

	// Check for authorization errors (notification doesn't belong to user)
	case err != nil && err.Error() == "notification does not belong to user":
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"You do not have permission to access this notification",
		)

	default:
		// Unknown error - return generic 500 without exposing internal details
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred. Please try again later.",
		)
	}
}
