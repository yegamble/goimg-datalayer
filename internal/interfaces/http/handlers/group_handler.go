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

func (h *GroupHandler) PublicRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListPublicGroups)
	r.Get("/search", h.SearchGroups)
	r.Get("/{groupID}", h.GetGroup)
	r.Get("/by-slug/{slug}", h.GetGroupBySlug)

	return r
}

func (h *GroupHandler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateGroup)
	r.Put("/{groupID}", h.UpdateGroup)
	r.Delete("/{groupID}", h.DeleteGroup)

	r.Post("/{groupID}/join", h.JoinGroup)
	r.Delete("/{groupID}/leave", h.LeaveGroup)
	r.Get("/{groupID}/members", h.ListMembers)

	r.Put("/{groupID}/members/{userID}/role", h.UpdateMemberRole)
	r.Delete("/{groupID}/members/{userID}", h.RemoveMember)
	r.Post("/{groupID}/members/{userID}/ban", h.BanMember)

	r.Post("/{groupID}/invitations", h.CreateInvitation)
	r.Get("/{groupID}/invitations", h.ListInvitations)
	r.Post("/invitations/{token}/accept", h.AcceptInvitation)
	r.Post("/invitations/{token}/decline", h.DeclineInvitation)

	return r
}

func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in create group handler")
		middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	if !userCtx.EmailVerified {
		middleware.WriteError(w, r, http.StatusForbidden, "Forbidden", "Email verification required to create groups")
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

	err = h.deleteGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "delete group")
		return
	}

	h.logger.Info().Str("group_id", groupIDStr).Str("owner_id", userCtx.UserID.String()).Msg("group deleted successfully")
	w.WriteHeader(http.StatusNoContent)
}

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

	err = h.leaveGroup.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "leave group")
		return
	}

	h.logger.Info().Str("group_id", groupIDStr).Str("user_id", userCtx.UserID.String()).Msg("user left group successfully")
	w.WriteHeader(http.StatusNoContent)
}

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

	err = h.removeMember.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "remove member")
		return
	}

	h.logger.Info().Str("group_id", groupIDStr).Str("actor_id", userCtx.UserID.String()).Str("target_id", targetIDStr).Msg("member removed successfully")
	w.WriteHeader(http.StatusNoContent)
}

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

	err = h.banMember.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "ban member")
		return
	}

	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Str("target_id", targetIDStr).
		Msg("member banned successfully")

	w.WriteHeader(http.StatusNoContent)
}

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

func (h *GroupHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("group operation failed")

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
