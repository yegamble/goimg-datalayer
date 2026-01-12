package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/community/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/community/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

const (
	// defaultPerPage is the default number of items returned per page in list/search endpoints.
	defaultPerPage = 20
	// maxPerPage is the maximum number of items allowed per page.
	maxPerPage = 100
)

// GroupHandler handles group-related HTTP endpoints.
// It delegates to application layer command and query handlers for business logic.
type GroupHandler struct {
	createGroup       *commands.CreateGroupHandler
	updateGroup       *commands.UpdateGroupHandler
	deleteGroup       *commands.DeleteGroupHandler
	joinGroup         *commands.JoinGroupHandler
	leaveGroup        *commands.LeaveGroupHandler
	updateMemberRole  *commands.UpdateMemberRoleHandler
	removeMember      *commands.RemoveMemberHandler
	banMember         *commands.BanMemberHandler
	getGroup          *queries.GetGroupHandler
	getGroupBySlug    *queries.GetGroupBySlugHandler
	listPublicGroups  *queries.ListPublicGroupsHandler
	searchGroups      *queries.SearchGroupsHandler
	listGroupMembers  *queries.ListGroupMembersHandler
	listUserGroups    *queries.ListUserGroupsHandler
	logger            zerolog.Logger
}

// NewGroupHandler creates a new GroupHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewGroupHandler(
	createGroup *commands.CreateGroupHandler,
	updateGroup *commands.UpdateGroupHandler,
	deleteGroup *commands.DeleteGroupHandler,
	joinGroup *commands.JoinGroupHandler,
	leaveGroup *commands.LeaveGroupHandler,
	updateMemberRole *commands.UpdateMemberRoleHandler,
	removeMember *commands.RemoveMemberHandler,
	banMember *commands.BanMemberHandler,
	getGroup *queries.GetGroupHandler,
	getGroupBySlug *queries.GetGroupBySlugHandler,
	listPublicGroups *queries.ListPublicGroupsHandler,
	searchGroups *queries.SearchGroupsHandler,
	listGroupMembers *queries.ListGroupMembersHandler,
	listUserGroups *queries.ListUserGroupsHandler,
	logger zerolog.Logger,
) *GroupHandler {
	return &GroupHandler{
		createGroup:      createGroup,
		updateGroup:      updateGroup,
		deleteGroup:      deleteGroup,
		joinGroup:        joinGroup,
		leaveGroup:       leaveGroup,
		updateMemberRole: updateMemberRole,
		removeMember:     removeMember,
		banMember:        banMember,
		getGroup:         getGroup,
		getGroupBySlug:   getGroupBySlug,
		listPublicGroups: listPublicGroups,
		searchGroups:     searchGroups,
		listGroupMembers: listGroupMembers,
		listUserGroups:   listUserGroups,
		logger:           logger,
	}
}

// PublicRoutes returns group routes that don't require authentication.
// These can be mounted directly under /api/v1/groups in the public section.
func (h *GroupHandler) PublicRoutes() chi.Router {
	r := chi.NewRouter()

	// Public routes (no authentication required)
	r.Get("/", h.ListPublicGroups)         // List discoverable groups
	r.Get("/search", h.SearchGroups)       // Search groups
	r.Get("/{groupID}", h.GetGroup)        // Get group by ID
	r.Get("/by-slug/{slug}", h.GetGroupBySlug) // Get group by slug

	return r
}

// ProtectedRoutes returns group routes that require authentication.
// These should be mounted under /api/v1/groups in the protected section.
func (h *GroupHandler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()

	// Group CRUD routes (require authentication)
	r.Post("/", h.CreateGroup)             // Create group
	r.Put("/{groupID}", h.UpdateGroup)     // Update group (admin+)
	r.Delete("/{groupID}", h.DeleteGroup)  // Delete group (owner only)

	// Membership routes (require authentication)
	r.Post("/{groupID}/join", h.JoinGroup)      // Join group
	r.Delete("/{groupID}/leave", h.LeaveGroup)  // Leave group
	r.Get("/{groupID}/members", h.ListMembers)  // List group members

	// Member management routes (admin+ only)
	r.Put("/{groupID}/members/{userID}/role", h.UpdateMemberRole) // Update role
	r.Delete("/{groupID}/members/{userID}", h.RemoveMember)       // Remove member
	r.Post("/{groupID}/members/{userID}/ban", h.BanMember)        // Ban member

	return r
}

// CreateGroup handles POST /api/v1/groups
// Creates a new group with the authenticated user as owner.
//
// Request: CreateGroupRequest JSON body
// Response: 201 Created with GroupResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 409: Group slug already taken
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with validation and response.
func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in create group handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Decode request body
	var req CreateGroupRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid create group request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid group data",
			validationErrors,
		)
		return
	}

	// 3. Build create command
	cmd := commands.CreateGroupCommand{
		OwnerID:     identity.UserID{}, // Will be set from userCtx
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		GroupType:   req.GroupType,
		Settings:    req.Settings,
	}

	// Parse owner ID from context
	ownerID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}
	cmd.OwnerID = ownerID

	// 4. Execute create command
	group, err := h.createGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "create group")
		return
	}

	// 5. Map to response DTO
	resp := mapGroupToResponse(group)

	h.logger.Info().
		Str("group_id", group.ID().String()).
		Str("slug", group.Slug().String()).
		Str("owner_id", userCtx.UserID.String()).
		Msg("group created successfully")

	if err := EncodeJSON(w, http.StatusCreated, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode create group response")
	}
}

// GetGroup handles GET /api/v1/groups/{groupID}
// Retrieves a single group by its ID.
//
// Path parameters:
//   - groupID: UUID of the group
//
// Response: 200 OK with GroupResponse
// Errors:
//   - 400: Invalid group ID format
//   - 404: Group not found
//   - 500: Internal server error
func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 2. Build query
	query := queries.GetGroupQuery{
		GroupID: groupID,
	}

	// 3. Execute query
	group, err := h.getGroup.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get group")
		return
	}

	// 4. Return group DTO
	resp := mapGroupToResponse(group)

	h.logger.Debug().
		Str("group_id", groupIDStr).
		Msg("group retrieved successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get group response")
	}
}

// GetGroupBySlug handles GET /api/v1/groups/by-slug/{slug}
// Retrieves a single group by its slug.
//
// Path parameters:
//   - slug: URL-friendly slug of the group
//
// Response: 200 OK with GroupResponse
// Errors:
//   - 400: Invalid or missing slug
//   - 404: Group not found
//   - 500: Internal server error
func (h *GroupHandler) GetGroupBySlug(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract slug from path
	slugStr := GetPathParam(r, "slug")
	if slugStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group slug",
		)
		return
	}

	slug, err := community.NewGroupSlug(slugStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("slug", slugStr).Msg("invalid group slug")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group slug format",
		)
		return
	}

	// 2. Build query
	query := queries.GetGroupBySlugQuery{
		Slug: slug,
	}

	// 3. Execute query
	group, err := h.getGroupBySlug.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get group by slug")
		return
	}

	// 4. Return group DTO
	resp := mapGroupToResponse(group)

	h.logger.Debug().
		Str("slug", slugStr).
		Msg("group retrieved by slug successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get group by slug response")
	}
}

// UpdateGroup handles PUT /api/v1/groups/{groupID}
// Updates group metadata and settings (admin+ only).
//
// Path parameters:
//   - groupID: UUID of the group
//
// Request: UpdateGroupRequest JSON body
// Response: 200 OK with GroupResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: Insufficient permissions (not admin/owner)
//   - 404: Group not found
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with validation and response.
func (h *GroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in update group handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 3. Decode request body
	var req UpdateGroupRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid update group request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid group update data",
			validationErrors,
		)
		return
	}

	// 4. Parse actor ID
	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 5. Build update command
	cmd := commands.UpdateGroupCommand{
		GroupID:     groupID,
		ActorID:     actorID,
		Description: req.Description,
		Settings:    req.Settings,
	}

	// 6. Execute update command
	group, err := h.updateGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "update group")
		return
	}

	// 7. Return updated group
	resp := mapGroupToResponse(group)

	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Msg("group updated successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update group response")
	}
}

// DeleteGroup handles DELETE /api/v1/groups/{groupID}
// Deletes a group (owner only).
//
// Path parameters:
//   - groupID: UUID of the group
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid group ID format
//   - 401: Not authenticated
//   - 403: Not the owner
//   - 404: Group not found
//   - 500: Internal server error
func (h *GroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in delete group handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 3. Parse owner ID
	ownerID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 4. Build delete command
	cmd := commands.DeleteGroupCommand{
		GroupID: groupID,
		OwnerID: ownerID,
	}

	// 5. Execute delete command
	_, err = h.deleteGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "delete group")
		return
	}

	// 6. Return 204 No Content
	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("owner_id", userCtx.UserID.String()).
		Msg("group deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}

// ListPublicGroups handles GET /api/v1/groups
// Lists public and invite-only groups with filtering and pagination.
//
// Query parameters:
//   - group_type: Filter by group type: public, private, invite-only (optional)
//   - sort_by: Sort order: recent, popular, name, activity (default: recent)
//   - page: Page number, 1-indexed (default: 1)
//   - per_page: Items per page, max 100 (default: 20)
//
// Response: 200 OK with PaginatedGroupsResponse
// Errors:
//   - 400: Invalid query parameters
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with query parameter parsing.
func (h *GroupHandler) ListPublicGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Parse query parameters
	queryParams := r.URL.Query()

	var groupType *community.GroupType
	if gt := queryParams.Get("group_type"); gt != "" {
		parsed, err := community.ParseGroupType(gt)
		if err != nil {
			middleware.WriteError(w, r,
				http.StatusBadRequest,
				"Bad Request",
				"Invalid group_type parameter",
			)
			return
		}
		groupType = &parsed
	}

	sortByStr := queryParams.Get("sort_by")
	if sortByStr == "" {
		sortByStr = "recent"
	}
	sortBy, err := community.ParseGroupSortBy(sortByStr)
	if err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid sort_by parameter",
		)
		return
	}

	page, err := parseIntParam(queryParams.Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	// 2. Build list query
	query := queries.ListPublicGroupsQuery{
		GroupType: groupType,
		SortBy:    sortBy,
		Page:      page,
		PerPage:   perPage,
	}

	// 3. Execute query
	result, err := h.listPublicGroups.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list public groups")
		return
	}

	// 4. Return paginated results
	resp := PaginatedGroupsResponse{
		Groups:     mapGroupsToResponse(result.Groups),
		TotalCount: int64(result.Total),
		Page:       page,
		PerPage:    perPage,
		TotalPages: int64((result.Total + perPage - 1) / perPage),
	}

	h.logger.Debug().
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Groups)).
		Int("total", result.Total).
		Msg("public groups listed successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list public groups response")
	}
}

// SearchGroups handles GET /api/v1/groups/search
// Full-text search for groups by name or description.
//
// Query parameters:
//   - q: Search query (required)
//   - page: Page number, 1-indexed (default: 1)
//   - per_page: Items per page, max 100 (default: 20)
//
// Response: 200 OK with SearchGroupsResponse
// Errors:
//   - 400: Invalid query parameters or missing search query
//   - 500: Internal server error
func (h *GroupHandler) SearchGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Parse query parameters
	queryParams := r.URL.Query()

	searchQuery := queryParams.Get("q")
	if searchQuery == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Search query (q) is required",
		)
		return
	}

	page, err := parseIntParam(queryParams.Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	// 2. Build search query
	query := queries.SearchGroupsQuery{
		Query:   searchQuery,
		Page:    page,
		PerPage: perPage,
	}

	// 3. Execute search
	result, err := h.searchGroups.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "search groups")
		return
	}

	// 4. Return search results
	resp := SearchGroupsResponse{
		Groups:     mapGroupsToResponse(result.Groups),
		TotalCount: int64(result.Total),
		Page:       page,
		PerPage:    perPage,
		TotalPages: int64((result.Total + perPage - 1) / perPage),
		Query:      result.Query,
	}

	h.logger.Debug().
		Str("query", searchQuery).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Groups)).
		Int("total", result.Total).
		Msg("group search completed")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode search groups response")
	}
}

// JoinGroup handles POST /api/v1/groups/{groupID}/join
// Allows a user to join a group (behavior depends on group type).
//
// Path parameters:
//   - groupID: UUID of the group
//
// Response: 201 Created with MembershipResponse
// Errors:
//   - 400: Invalid group ID
//   - 401: Not authenticated
//   - 403: Cannot join private group or already banned
//   - 404: Group not found
//   - 409: Already a member
//   - 500: Internal server error
func (h *GroupHandler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in join group handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 3. Parse user ID
	userID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 4. Build join command
	cmd := commands.JoinGroupCommand{
		GroupID: groupID,
		UserID:  userID,
	}

	// 5. Execute join command
	membership, err := h.joinGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "join group")
		return
	}

	// 6. Return membership response
	resp := mapMembershipToResponse(membership)

	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("user_id", userCtx.UserID.String()).
		Str("status", membership.Status().String()).
		Msg("user joined group successfully")

	if err := EncodeJSON(w, http.StatusCreated, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode join group response")
	}
}

// LeaveGroup handles DELETE /api/v1/groups/{groupID}/leave
// Allows a user to leave a group (owner cannot leave).
//
// Path parameters:
//   - groupID: UUID of the group
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid group ID
//   - 401: Not authenticated
//   - 403: Owner cannot leave group
//   - 404: Group not found or not a member
//   - 500: Internal server error
func (h *GroupHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in leave group handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 3. Parse user ID
	userID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 4. Build leave command
	cmd := commands.LeaveGroupCommand{
		GroupID: groupID,
		UserID:  userID,
	}

	// 5. Execute leave command
	_, err = h.leaveGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "leave group")
		return
	}

	// 6. Return 204 No Content
	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("user_id", userCtx.UserID.String()).
		Msg("user left group successfully")

	w.WriteHeader(http.StatusNoContent)
}

// ListMembers handles GET /api/v1/groups/{groupID}/members
// Lists members of a group with optional filtering.
//
// Path parameters:
//   - groupID: UUID of the group
//
// Query parameters:
//   - role: Filter by role: owner, admin, moderator, member (optional)
//   - status: Filter by status: active, invited, requested, banned (optional)
//   - page: Page number, 1-indexed (default: 1)
//   - per_page: Items per page, max 100 (default: 20)
//
// Response: 200 OK with PaginatedMembersResponse
// Errors:
//   - 400: Invalid parameters
//   - 404: Group not found
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with query parameter parsing.
func (h *GroupHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 2. Parse query parameters
	queryParams := r.URL.Query()

	var role *community.GroupRole
	if roleStr := queryParams.Get("role"); roleStr != "" {
		parsed, err := community.ParseGroupRole(roleStr)
		if err != nil {
			middleware.WriteError(w, r,
				http.StatusBadRequest,
				"Bad Request",
				"Invalid role parameter",
			)
			return
		}
		role = &parsed
	}

	var status *community.MemberStatus
	if statusStr := queryParams.Get("status"); statusStr != "" {
		parsed, err := community.ParseMemberStatus(statusStr)
		if err != nil {
			middleware.WriteError(w, r,
				http.StatusBadRequest,
				"Bad Request",
				"Invalid status parameter",
			)
			return
		}
		status = &parsed
	}

	page, err := parseIntParam(queryParams.Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	// 3. Build query
	query := queries.ListGroupMembersQuery{
		GroupID: groupID,
		Role:    role,
		Status:  status,
		Page:    page,
		PerPage: perPage,
	}

	// 4. Execute query
	result, err := h.listGroupMembers.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list group members")
		return
	}

	// 5. Return paginated results
	resp := PaginatedMembersResponse{
		Members:    mapMembershipsToResponse(result.Members),
		TotalCount: int64(result.Total),
		Page:       page,
		PerPage:    perPage,
		TotalPages: int64((result.Total + perPage - 1) / perPage),
	}

	h.logger.Debug().
		Str("group_id", groupIDStr).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Members)).
		Int("total", result.Total).
		Msg("group members listed successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list members response")
	}
}

// UpdateMemberRole handles PUT /api/v1/groups/{groupID}/members/{userID}/role
// Updates a member's role in the group (admin+ only).
//
// Path parameters:
//   - groupID: UUID of the group
//   - userID: UUID of the member
//
// Request: UpdateMemberRoleRequest JSON body
// Response: 200 OK with MembershipResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: Insufficient permissions
//   - 404: Group or member not found
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with validation and response.
func (h *GroupHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in update member role handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 3. Extract target user ID from path
	targetIDStr := GetPathParam(r, "userID")
	if targetIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	targetID, err := identity.ParseUserID(targetIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("user_id", targetIDStr).Msg("invalid user id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid user ID format",
		)
		return
	}

	// 4. Decode request body
	var req UpdateMemberRoleRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid update member role request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid role update data",
			validationErrors,
		)
		return
	}

	// 5. Parse role
	newRole, err := community.ParseGroupRole(req.Role)
	if err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid role value",
		)
		return
	}

	// 6. Parse actor ID
	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 7. Build command
	cmd := commands.UpdateMemberRoleCommand{
		GroupID:  groupID,
		ActorID:  actorID,
		TargetID: targetID,
		NewRole:  newRole,
	}

	// 8. Execute command
	membership, err := h.updateMemberRole.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "update member role")
		return
	}

	// 9. Return updated membership
	resp := mapMembershipToResponse(membership)

	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Str("target_id", targetIDStr).
		Str("new_role", newRole.String()).
		Msg("member role updated successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update member role response")
	}
}

// RemoveMember handles DELETE /api/v1/groups/{groupID}/members/{userID}
// Removes a member from the group (admin+ only).
//
// Path parameters:
//   - groupID: UUID of the group
//   - userID: UUID of the member to remove
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid parameters
//   - 401: Not authenticated
//   - 403: Insufficient permissions or cannot remove owner
//   - 404: Group or member not found
//   - 500: Internal server error
func (h *GroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in remove member handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 3. Extract target user ID from path
	targetIDStr := GetPathParam(r, "userID")
	if targetIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	targetID, err := identity.ParseUserID(targetIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("user_id", targetIDStr).Msg("invalid user id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid user ID format",
		)
		return
	}

	// 4. Parse actor ID
	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 5. Build command
	cmd := commands.RemoveMemberCommand{
		GroupID:  groupID,
		ActorID:  actorID,
		TargetID: targetID,
	}

	// 6. Execute command
	_, err = h.removeMember.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "remove member")
		return
	}

	// 7. Return 204 No Content
	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Str("target_id", targetIDStr).
		Msg("member removed successfully")

	w.WriteHeader(http.StatusNoContent)
}

// BanMember handles POST /api/v1/groups/{groupID}/members/{userID}/ban
// Bans a member from the group (admin+ only).
//
// Path parameters:
//   - groupID: UUID of the group
//   - userID: UUID of the member to ban
//
// Request: BanMemberRequest JSON body
// Response: 200 OK with MembershipResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: Insufficient permissions or cannot ban owner/admin
//   - 404: Group or member not found
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with validation and response.
func (h *GroupHandler) BanMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in ban member handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 3. Extract target user ID from path
	targetIDStr := GetPathParam(r, "userID")
	if targetIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	targetID, err := identity.ParseUserID(targetIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("user_id", targetIDStr).Msg("invalid user id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid user ID format",
		)
		return
	}

	// 4. Decode request body
	var req BanMemberRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid ban member request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid ban data",
			validationErrors,
		)
		return
	}

	// 5. Parse actor ID
	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 6. Build command
	cmd := commands.BanMemberCommand{
		GroupID:  groupID,
		ActorID:  actorID,
		TargetID: targetID,
		Reason:   req.Reason,
	}

	// 7. Execute command
	membership, err := h.banMember.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "ban member")
		return
	}

	// 8. Return updated membership
	resp := mapMembershipToResponse(membership)

	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Str("target_id", targetIDStr).
		Msg("member banned successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode ban member response")
	}
}

// GetUserGroups is a convenience method for mounting under /api/v1/me/groups.
// It delegates to ListUserGroups with the current user's ID.
func (h *GroupHandler) GetUserGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in get user groups handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Parse query parameters
	queryParams := r.URL.Query()

	page, err := parseIntParam(queryParams.Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	// 3. Parse user ID
	userID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 4. Build query
	query := queries.ListUserGroupsQuery{
		UserID:  userID,
		Page:    page,
		PerPage: perPage,
	}

	// 5. Execute query
	result, err := h.listUserGroups.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list user groups")
		return
	}

	// 6. Return paginated results
	resp := PaginatedUserGroupsResponse{
		Memberships: mapMembershipsToResponse(result.Memberships),
		TotalCount:  int64(result.Total),
		Page:        page,
		PerPage:     perPage,
		TotalPages:  int64((result.Total + perPage - 1) / perPage),
	}

	h.logger.Debug().
		Str("user_id", userCtx.UserID.String()).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Memberships)).
		Int("total", result.Total).
		Msg("user groups listed successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode user groups response")
	}
}

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
func (h *GroupHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("group operation failed")

	// Map domain errors to HTTP status codes
	switch {
	case err == community.ErrGroupNotFound:
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Group not found",
		)
	case err == community.ErrGroupSlugTaken:
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"Group slug is already taken",
		)
	case err == community.ErrInsufficientGroupRole:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Insufficient permissions for this operation",
		)
	case err == community.ErrAlreadyGroupMember:
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"User is already a member of this group",
		)
	case err == community.ErrPrivateGroupNoAccess:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Cannot join private group without invitation",
		)
	case err == community.ErrMemberLimitReached:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Group has reached maximum member capacity",
		)
	case err == community.ErrMemberBanned:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"User is banned from this group",
		)
	case err == community.ErrMembershipNotFound:
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Membership not found",
		)
	case err == community.ErrCannotRemoveOwner:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Cannot remove the group owner",
		)
	case err == community.ErrCannotBanOwner:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Cannot ban the group owner",
		)
	case err == community.ErrOwnerCannotLeave:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Owner cannot leave the group",
		)
	default:
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred",
		)
	}
}

// Helper functions to map domain entities to DTOs
func mapGroupToResponse(group *community.Group) GroupResponse {
	resp := GroupResponse{
		ID:          group.ID().String(),
		Name:        group.Name().String(),
		Slug:        group.Slug().String(),
		Description: group.Description(),
		GroupType:   group.GroupType().String(),
		OwnerID:     group.OwnerID().String(),
		Settings:    mapSettingsToResponse(group.Settings()),
		MemberCount: group.MemberCount(),
		ImageCount:  group.ImageCount(),
		AlbumCount:  group.AlbumCount(),
		CreatedAt:   group.CreatedAt(),
		UpdatedAt:   group.UpdatedAt(),
	}

	if coverID := group.CoverImageID(); coverID != nil {
		coverIDStr := coverID.String()
		resp.CoverImageID = &coverIDStr
	}

	return resp
}

func mapGroupsToResponse(groups []*community.Group) []GroupResponse {
	resp := make([]GroupResponse, len(groups))
	for i, group := range groups {
		resp[i] = mapGroupToResponse(group)
	}
	return resp
}

func mapMembershipToResponse(membership *community.GroupMembership) MembershipResponse {
	return MembershipResponse{
		ID:        membership.ID().String(),
		GroupID:   membership.GroupID().String(),
		UserID:    membership.UserID().String(),
		Role:      membership.Role().String(),
		Status:    membership.Status().String(),
		JoinedAt:  membership.JoinedAt(),
		UpdatedAt: membership.UpdatedAt(),
	}
}

func mapMembershipsToResponse(memberships []*community.GroupMembership) []MembershipResponse {
	resp := make([]MembershipResponse, len(memberships))
	for i, membership := range memberships {
		resp[i] = mapMembershipToResponse(membership)
	}
	return resp
}

func mapSettingsToResponse(settings community.GroupSettings) GroupSettingsResponse {
	return GroupSettingsResponse{
		MaxMembers:          settings.MaxMembers(),
		RequireApproval:     settings.RequireApproval(),
		AllowGuestUploads:   settings.AllowGuestUploads(),
		AllowComments:       settings.AllowComments(),
		DefaultImagePrivacy: settings.DefaultImagePrivacy().String(),
	}
}

// Helper function from image_handler.go - parseIntParam
// Already defined in existing handlers, but including here for completeness
// func parseIntParam(param string, defaultValue int) (int, error) { ... }
