package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/community/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/community/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// GroupHandler handles group-related HTTP endpoints.
// It delegates to application layer command and query handlers for business logic.
type GroupHandler struct {
	createGroup          *commands.CreateGroupHandler
	updateGroup          *commands.UpdateGroupHandler
	deleteGroup          *commands.DeleteGroupHandler
	joinGroup            *commands.JoinGroupHandler
	leaveGroup           *commands.LeaveGroupHandler
	updateMemberRole     *commands.UpdateMemberRoleHandler
	removeMember         *commands.RemoveMemberHandler
	banMember            *commands.BanMemberHandler
	inviteToGroup        *commands.InviteToGroupHandler
	acceptInvitation     *commands.AcceptInvitationHandler
	declineInvitation    *commands.DeclineInvitationHandler
	getGroup             *queries.GetGroupHandler
	getGroupBySlug       *queries.GetGroupBySlugHandler
	listPublicGroups     *queries.ListPublicGroupsHandler
	searchGroups         *queries.SearchGroupsHandler
	listGroupMembers     *queries.ListGroupMembersHandler
	listUserGroups       *queries.ListUserGroupsHandler
	listGroupInvitations *queries.ListGroupInvitationsHandler
	logger               zerolog.Logger
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
	inviteToGroup *commands.InviteToGroupHandler,
	acceptInvitation *commands.AcceptInvitationHandler,
	declineInvitation *commands.DeclineInvitationHandler,
	getGroup *queries.GetGroupHandler,
	getGroupBySlug *queries.GetGroupBySlugHandler,
	listPublicGroups *queries.ListPublicGroupsHandler,
	searchGroups *queries.SearchGroupsHandler,
	listGroupMembers *queries.ListGroupMembersHandler,
	listUserGroups *queries.ListUserGroupsHandler,
	listGroupInvitations *queries.ListGroupInvitationsHandler,
	logger zerolog.Logger,
) *GroupHandler {
	return &GroupHandler{
		createGroup:          createGroup,
		updateGroup:          updateGroup,
		deleteGroup:          deleteGroup,
		joinGroup:            joinGroup,
		leaveGroup:           leaveGroup,
		updateMemberRole:     updateMemberRole,
		removeMember:         removeMember,
		banMember:            banMember,
		inviteToGroup:        inviteToGroup,
		acceptInvitation:     acceptInvitation,
		declineInvitation:    declineInvitation,
		getGroup:             getGroup,
		getGroupBySlug:       getGroupBySlug,
		listPublicGroups:     listPublicGroups,
		searchGroups:         searchGroups,
		listGroupMembers:     listGroupMembers,
		listUserGroups:       listUserGroups,
		listGroupInvitations: listGroupInvitations,
		logger:               logger,
	}
}

// PublicRoutes returns group routes that don't require authentication.
// These can be mounted directly under /api/v1/groups in the public section.
func (h *GroupHandler) PublicRoutes() chi.Router {
	r := chi.NewRouter()

	// Public routes (no authentication required)
	r.Get("/", h.ListPublicGroups)             // List discoverable groups
	r.Get("/search", h.SearchGroups)           // Search groups
	r.Get("/{groupID}", h.GetGroup)            // Get group by ID
	r.Get("/by-slug/{slug}", h.GetGroupBySlug) // Get group by slug

	return r
}

// ProtectedRoutes returns group routes that require authentication.
// These should be mounted under /api/v1/groups in the protected section.
func (h *GroupHandler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()

	// Group CRUD routes (require authentication)
	r.Post("/", h.CreateGroup)            // Create group
	r.Put("/{groupID}", h.UpdateGroup)    // Update group (admin+)
	r.Delete("/{groupID}", h.DeleteGroup) // Delete group (owner only)

	// Membership routes (require authentication)
	r.Post("/{groupID}/join", h.JoinGroup)     // Join group
	r.Delete("/{groupID}/leave", h.LeaveGroup) // Leave group
	r.Get("/{groupID}/members", h.ListMembers) // List group members

	// Member management routes (admin+ only)
	r.Put("/{groupID}/members/{userID}/role", h.UpdateMemberRole) // Update role
	r.Delete("/{groupID}/members/{userID}", h.RemoveMember)       // Remove member
	r.Post("/{groupID}/members/{userID}/ban", h.BanMember)        // Ban member

	// Invitation routes
	r.Post("/{groupID}/invitations", h.CreateInvitation)        // Create invitation (admin/owner+)
	r.Get("/{groupID}/invitations", h.ListInvitations)          // List pending invitations (admin/owner+)
	r.Post("/invitations/{token}/accept", h.AcceptInvitation)   // Accept invitation (any auth user)
	r.Post("/invitations/{token}/decline", h.DeclineInvitation) // Decline invitation (any auth user)

	return r
}

// CreateGroup handles POST /api/v1/groups
func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in create group handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	var req CreateGroupRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid create group request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r, http.StatusBadRequest, "Validation Failed", "Invalid group data", validationErrors)
		return
	}

	ownerID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.CreateGroupCommand{
		OwnerID:     ownerID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		GroupType:   req.GroupType,
		Settings:    req.Settings,
	}

	group, err := h.createGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "create group")
		return
	}

	resp := mapGroupToResponse(group)
	h.logger.Info().Str("group_id", group.ID().String()).Str("slug", group.Slug().String()).Str("owner_id", userCtx.UserID.String()).Msg("group created successfully")
	if err := EncodeJSON(w, http.StatusCreated, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode create group response")
	}
}

// GetGroup handles GET /api/v1/groups/{groupID}
func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Missing group ID")
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID format")
		return
	}

	query := queries.GetGroupQuery{GroupID: groupID}
	group, err := h.getGroup.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get group")
		return
	}

	resp := mapGroupToResponse(group)
	h.logger.Debug().Str("group_id", groupIDStr).Msg("group retrieved successfully")
	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get group response")
	}
}

// GetGroupBySlug handles GET /api/v1/groups/by-slug/{slug}
func (h *GroupHandler) GetGroupBySlug(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slugStr := GetPathParam(r, "slug")
	if slugStr == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Missing group slug")
		return
	}

	slug, err := community.NewGroupSlug(slugStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("slug", slugStr).Msg("invalid group slug")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group slug format")
		return
	}

	query := queries.GetGroupBySlugQuery{Slug: slug}
	group, err := h.getGroupBySlug.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get group by slug")
		return
	}

	resp := mapGroupToResponse(group)
	h.logger.Debug().Str("slug", slugStr).Msg("group retrieved by slug successfully")
	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get group by slug response")
	}
}

// UpdateGroup handles PUT /api/v1/groups/{groupID}
func (h *GroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in update group handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID format")
		return
	}

	var req UpdateGroupRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid update group request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r, http.StatusBadRequest, "Validation Failed", "Invalid group update data", validationErrors)
		return
	}

	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.UpdateGroupCommand{
		GroupID:     groupID,
		ActorID:     actorID,
		Description: req.Description,
		Settings:    req.Settings,
	}

	group, err := h.updateGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "update group")
		return
	}

	resp := mapGroupToResponse(group)
	h.logger.Info().Str("group_id", groupIDStr).Str("actor_id", userCtx.UserID.String()).Msg("group updated successfully")
	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update group response")
	}
}

// DeleteGroup handles DELETE /api/v1/groups/{groupID}
func (h *GroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in delete group handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID format")
		return
	}

	ownerID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.DeleteGroupCommand{
		GroupID: groupID,
		OwnerID: ownerID,
	}

	// 5. Execute delete command
	err = h.deleteGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "delete group")
		return
	}

	h.logger.Info().Str("group_id", groupIDStr).Str("owner_id", userCtx.UserID.String()).Msg("group deleted successfully")
	w.WriteHeader(http.StatusNoContent)
}

// ListPublicGroups handles GET /api/v1/groups
func (h *GroupHandler) ListPublicGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryParams := r.URL.Query()

	var groupType *community.GroupType
	if gt := queryParams.Get("group_type"); gt != "" {
		parsed, err := community.ParseGroupType(gt)
		if err != nil {
			middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group_type parameter")
			return
		}
		groupType = &parsed
	}

	sortByStr := queryParams.Get("sort_by")
	if sortByStr == "" {
		sortByStr = "recent"
	}
	var sortBy community.GroupSortBy
	switch sortByStr {
	case "recent":
		sortBy = community.GroupSortByRecent
	case "popular":
		sortBy = community.GroupSortByPopular
	case "name":
		sortBy = community.GroupSortByName
	case "activity":
		sortBy = community.GroupSortByActivity
	default:
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid sort_by parameter",
		)
		return
	}

	page, _ := parseIntParam(queryParams.Get("page"), 1)
	if page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultLimit)
	if err != nil || perPage < 1 {
		perPage = defaultLimit
	}
	if perPage > maxLimit {
		perPage = maxLimit
	}

	query := queries.ListPublicGroupsQuery{
		GroupType: groupType,
		SortBy:    sortBy,
		Page:      page,
		PerPage:   perPage,
	}

	result, err := h.listPublicGroups.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list public groups")
		return
	}

	resp := PaginatedGroupsResponse{
		Groups:     mapGroupsToResponse(result.Groups),
		TotalCount: int64(result.Total),
		Page:       page,
		PerPage:    perPage,
		TotalPages: int64((result.Total + perPage - 1) / perPage),
	}

	h.logger.Debug().Int("page", page).Int("per_page", perPage).Int("results", len(result.Groups)).Int("total", result.Total).Msg("public groups listed successfully")
	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list public groups response")
	}
}

// SearchGroups handles GET /api/v1/groups/search
func (h *GroupHandler) SearchGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryParams := r.URL.Query()

	searchQuery := queryParams.Get("q")
	if searchQuery == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Search query (q) is required")
		return
	}

	page, _ := parseIntParam(queryParams.Get("page"), 1)
	if page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultLimit)
	if err != nil || perPage < 1 {
		perPage = defaultLimit
	}
	if perPage > maxLimit {
		perPage = maxLimit
	}

	query := queries.SearchGroupsQuery{
		Query:   searchQuery,
		Page:    page,
		PerPage: perPage,
	}

	result, err := h.searchGroups.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "search groups")
		return
	}

	resp := SearchGroupsResponse{
		Groups:     mapGroupsToResponse(result.Groups),
		TotalCount: int64(result.Total),
		Page:       page,
		PerPage:    perPage,
		TotalPages: int64((result.Total + perPage - 1) / perPage),
		Query:      result.Query,
	}

	h.logger.Debug().Str("query", searchQuery).Int("page", page).Int("per_page", perPage).Int("results", len(result.Groups)).Int("total", result.Total).Msg("group search completed")
	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode search groups response")
	}
}

// JoinGroup handles POST /api/v1/groups/{groupID}/join
func (h *GroupHandler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in join group handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID format")
		return
	}

	userID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.JoinGroupCommand{
		GroupID: groupID,
		UserID:  userID,
	}

	membership, err := h.joinGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "join group")
		return
	}

	resp := mapMembershipToResponse(membership)
	h.logger.Info().Str("group_id", groupIDStr).Str("user_id", userCtx.UserID.String()).Str("status", membership.Status().String()).Msg("user joined group successfully")
	if err := EncodeJSON(w, http.StatusCreated, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode join group response")
	}
}

// LeaveGroup handles DELETE /api/v1/groups/{groupID}/leave
func (h *GroupHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in leave group handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID format")
		return
	}

	userID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.LeaveGroupCommand{
		GroupID: groupID,
		UserID:  userID,
	}

	// 5. Execute leave command
	err = h.leaveGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "leave group")
		return
	}

	h.logger.Info().Str("group_id", groupIDStr).Str("user_id", userCtx.UserID.String()).Msg("user left group successfully")
	w.WriteHeader(http.StatusNoContent)
}

// ListMembers handles GET /api/v1/groups/{groupID}/members
func (h *GroupHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID format")
		return
	}

	queryParams := r.URL.Query()
	var role *community.GroupRole
	if roleStr := queryParams.Get("role"); roleStr != "" {
		parsed, err := community.ParseGroupRole(roleStr)
		if err != nil {
			middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid role parameter")
			return
		}
		role = &parsed
	}

	var status *community.MemberStatus
	if statusStr := queryParams.Get("status"); statusStr != "" {
		parsed, err := community.ParseMemberStatus(statusStr)
		if err != nil {
			middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid status parameter")
			return
		}
		status = &parsed
	}

	page, _ := parseIntParam(queryParams.Get("page"), 1)
	if page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultLimit)
	if err != nil || perPage < 1 {
		perPage = defaultLimit
	}
	if perPage > maxLimit {
		perPage = maxLimit
	}

	query := queries.ListGroupMembersQuery{
		GroupID: groupID,
		Role:    role,
		Status:  status,
		Page:    page,
		PerPage: perPage,
	}

	result, err := h.listGroupMembers.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list group members")
		return
	}

	resp := PaginatedMembersResponse{
		Members:    mapMembershipsToResponse(result.Members),
		TotalCount: int64(result.Total),
		Page:       page,
		PerPage:    perPage,
		TotalPages: int64((result.Total + perPage - 1) / perPage),
	}

	h.logger.Debug().Str("group_id", groupIDStr).Int("page", page).Int("per_page", perPage).Int("results", len(result.Members)).Int("total", result.Total).Msg("group members listed successfully")
	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list members response")
	}
}

// UpdateMemberRole handles PUT /api/v1/groups/{groupID}/members/{userID}/role
func (h *GroupHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in update member role handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID format")
		return
	}

	targetIDStr := GetPathParam(r, "userID")
	targetID, err := identity.ParseUserID(targetIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("user_id", targetIDStr).Msg("invalid user id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID format")
		return
	}

	var req UpdateMemberRoleRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid update member role request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r, http.StatusBadRequest, "Validation Failed", "Invalid role update data", validationErrors)
		return
	}

	newRole, err := community.ParseGroupRole(req.Role)
	if err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid role value")
		return
	}

	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.UpdateMemberRoleCommand{
		GroupID:  groupID,
		ActorID:  actorID,
		TargetID: targetID,
		NewRole:  newRole,
	}

	membership, err := h.updateMemberRole.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "update member role")
		return
	}

	resp := mapMembershipToResponse(membership)
	h.logger.Info().Str("group_id", groupIDStr).Str("actor_id", userCtx.UserID.String()).Str("target_id", targetIDStr).Str("new_role", newRole.String()).Msg("member role updated successfully")
	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update member role response")
	}
}

// RemoveMember handles DELETE /api/v1/groups/{groupID}/members/{userID}
func (h *GroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in remove member handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID format")
		return
	}

	targetIDStr := GetPathParam(r, "userID")
	targetID, err := identity.ParseUserID(targetIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("user_id", targetIDStr).Msg("invalid user id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID format")
		return
	}

	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.RemoveMemberCommand{
		GroupID:  groupID,
		ActorID:  actorID,
		TargetID: targetID,
	}

	// 6. Execute command
	err = h.removeMember.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "remove member")
		return
	}

	h.logger.Info().Str("group_id", groupIDStr).Str("actor_id", userCtx.UserID.String()).Str("target_id", targetIDStr).Msg("member removed successfully")
	w.WriteHeader(http.StatusNoContent)
}

// BanMember handles POST /api/v1/groups/{groupID}/members/{userID}/ban
func (h *GroupHandler) BanMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in ban member handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID format")
		return
	}

	targetIDStr := GetPathParam(r, "userID")
	targetID, err := identity.ParseUserID(targetIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("user_id", targetIDStr).Msg("invalid user id")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID format")
		return
	}

	var req BanMemberRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid ban member request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r, http.StatusBadRequest, "Validation Failed", "Invalid ban data", validationErrors)
		return
	}

	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.BanMemberCommand{
		GroupID:  groupID,
		ActorID:  actorID,
		TargetID: targetID,
		Reason:   req.Reason,
	}

	// 7. Execute command
	err = h.banMember.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "ban member")
		return
	}

	// 8. Return 204 No Content
	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Str("target_id", targetIDStr).
		Msg("member banned successfully")

	w.WriteHeader(http.StatusNoContent)
}

// GetUserGroups is a convenience method for mounting under /api/v1/me/groups.
func (h *GroupHandler) GetUserGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in get user groups handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	queryParams := r.URL.Query()
	page, _ := parseIntParam(queryParams.Get("page"), 1)
	if page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultLimit)
	if err != nil || perPage < 1 {
		perPage = defaultLimit
	}
	if perPage > maxLimit {
		perPage = maxLimit
	}

	userID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	query := queries.ListUserGroupsQuery{
		UserID:  userID,
		Page:    page,
		PerPage: perPage,
	}

	result, err := h.listUserGroups.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list user groups")
		return
	}

	resp := PaginatedUserGroupsResponse{
		Memberships: mapMembershipsToResponse(result.Memberships),
		TotalCount:  int64(result.Total),
		Page:        page,
		PerPage:     perPage,
		TotalPages:  int64((result.Total + perPage - 1) / perPage),
	}

	h.logger.Debug().Str("user_id", userCtx.UserID.String()).Int("page", page).Int("per_page", perPage).Int("results", len(result.Memberships)).Int("total", result.Total).Msg("user groups listed successfully")
	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode user groups response")
	}
}

// ============================================================================
// Invitation Handlers (Sprint 20)
// ============================================================================

// CreateInvitation handles POST /api/v1/groups/{groupID}/invitations
func (h *GroupHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID")
		return
	}

	var req InviteToGroupRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid invite request")
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid request body")
		return
	}

	// Validate exactly one of email/userID
	if (req.Email == nil && req.UserID == nil) || (req.Email != nil && req.UserID != nil) {
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Must provide either email or user_id")
		return
	}

	var userID *identity.UserID
	if req.UserID != nil {
		uid, err := identity.ParseUserID(*req.UserID)
		if err != nil {
			middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID")
			return
		}
		userID = &uid
	}

	inviterID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.InviteToGroupCommand{
		GroupID:   groupID,
		InvitedBy: inviterID,
		Email:     req.Email,
		UserID:    userID,
	}

	invitation, err := h.inviteToGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "invite_to_group")
		return
	}

	resp := mapInvitationToResponse(invitation)
	if err := EncodeJSON(w, http.StatusCreated, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode invitation response")
	}
}

// ListInvitations handles GET /api/v1/groups/{groupID}/invitations
func (h *GroupHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	groupIDStr := GetPathParam(r, "groupID")
	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid group ID")
		return
	}

	queryParams := r.URL.Query()
	page, _ := parseIntParam(queryParams.Get("page"), 1)
	if page < 1 {
		page = 1
	}
	perPage, _ := parseIntParam(queryParams.Get("per_page"), defaultPerPage)
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	query := queries.ListGroupInvitationsQuery{
		GroupID: groupID,
		Page:    page,
		PerPage: perPage,
	}

	result, err := h.listGroupInvitations.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list_invitations")
		return
	}

	resp := PaginatedInvitationsResponse{
		Invitations: mapInvitationsToResponse(result.Invitations),
		TotalCount:  result.Total,
	}

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode invitations response")
	}
}

// AcceptInvitation handles POST /api/v1/groups/invitations/{token}/accept
func (h *GroupHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	tokenStr := GetPathParam(r, "token")
	token, err := community.ParseInvitationToken(tokenStr)
	if err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid invitation token")
		return
	}

	userID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.AcceptInvitationCommand{
		Token:  token,
		UserID: userID,
	}

	membership, err := h.acceptInvitation.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "accept_invitation")
		return
	}

	resp := mapMembershipToResponse(membership)
	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode membership response")
	}
}

// DeclineInvitation handles POST /api/v1/groups/invitations/{token}/decline
func (h *GroupHandler) DeclineInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	tokenStr := GetPathParam(r, "token")
	token, err := community.ParseInvitationToken(tokenStr)
	if err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid invitation token")
		return
	}

	userID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Invalid user context")
		return
	}

	cmd := commands.DeclineInvitationCommand{
		Token:  token,
		UserID: userID,
	}

	err = h.declineInvitation.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "decline_invitation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
func (h *GroupHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("group operation failed")

	// Map domain errors to HTTP status codes
	switch {
	case errors.Is(err, community.ErrGroupNotFound):
		middleware.WriteError(w, r, http.StatusNotFound, "Not Found", "Group not found")
	case errors.Is(err, community.ErrGroupSlugTaken):
		middleware.WriteError(w, r, http.StatusConflict, "Conflict", "Group slug is already taken")
	case errors.Is(err, community.ErrInsufficientGroupRole):
		middleware.WriteError(w, r, http.StatusForbidden, "Forbidden", "Insufficient permissions for this operation")
	case errors.Is(err, community.ErrAlreadyGroupMember):
		middleware.WriteError(w, r, http.StatusConflict, "Conflict", "User is already a member of this group")
	case errors.Is(err, community.ErrPrivateGroupNoAccess):
		middleware.WriteError(w, r, http.StatusForbidden, "Forbidden", "Cannot join private group without invitation")
	case errors.Is(err, community.ErrMemberLimitReached):
		middleware.WriteError(w, r, http.StatusForbidden, "Forbidden", "Group has reached maximum member capacity")
	case errors.Is(err, community.ErrMemberBanned):
		middleware.WriteError(w, r, http.StatusForbidden, "Forbidden", "User is banned from this group")
	case errors.Is(err, community.ErrMembershipNotFound):
		middleware.WriteError(w, r, http.StatusNotFound, "Not Found", "Membership not found")
	case errors.Is(err, community.ErrCannotRemoveOwner):
		middleware.WriteError(w, r, http.StatusForbidden, "Forbidden", "Cannot remove the group owner")
	case errors.Is(err, community.ErrCannotBanOwner):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Cannot ban the group owner",
		)
	case errors.Is(err, community.ErrCannotLeaveAsOwner):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Owner cannot leave the group",
		)
	default:
		middleware.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
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
	if membership == nil {
		return MembershipResponse{}
	}
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
		MaxMembers:         settings.MaxMembers(),
		RequireApproval:    settings.RequireApproval(),
		AllowMemberInvites: settings.AllowMemberInvites(),
		AllowMemberAlbums:  settings.AllowMemberAlbums(),
	}
}

func mapInvitationToResponse(invitation *community.GroupInvitation) InvitationResponse {
	resp := InvitationResponse{
		ID:        invitation.ID().String(),
		GroupID:   invitation.GroupID().String(),
		InvitedBy: invitation.InvitedBy().String(),
		Token:     invitation.Token().String(),
		ExpiresAt: invitation.ExpiresAt(),
		CreatedAt: invitation.CreatedAt(),
	}

	if email := invitation.Email(); email != nil {
		resp.Email = email
	}

	if userID := invitation.UserID(); userID != nil {
		uidStr := userID.String()
		resp.UserID = &uidStr
	}

	if usedAt := invitation.UsedAt(); usedAt != nil {
		resp.UsedAt = usedAt
	}

	return resp
}

func mapInvitationsToResponse(invitations []*community.GroupInvitation) []InvitationResponse {
	resp := make([]InvitationResponse, len(invitations))
	for i, invitation := range invitations {
		resp[i] = mapInvitationToResponse(invitation)
	}
	return resp
}

func parseGroupSortBy(s string) (community.GroupSortBy, error) {
	switch s {
	case "recent":
		return community.GroupSortByRecent, nil
	case "popular":
		return community.GroupSortByPopular, nil
	case "name":
		return community.GroupSortByName, nil
	case "activity":
		return community.GroupSortByActivity, nil
	default:
		return "", fmt.Errorf("invalid sort_by parameter: %s", s)
	}
}
