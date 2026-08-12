package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	apporg "github.com/halcyon-io/halcyon/services/organization/internal/application/organization"
	httpmiddleware "github.com/halcyon-io/halcyon/services/organization/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

// OrganizationHandler serves organization HTTP endpoints.
type OrganizationHandler struct {
	create         *apporg.CreateOrganizationUseCase
	get            *apporg.GetOrganizationUseCase
	update         *apporg.UpdateOrganizationUseCase
	delete         *apporg.DeleteOrganizationUseCase
	listMembers    *apporg.ListMembersUseCase
	updateMember   *apporg.UpdateMemberUseCase
	removeMember   *apporg.RemoveMemberUseCase
	invite         *apporg.InviteMemberUseCase
	acceptInvite   *apporg.AcceptInvitationUseCase
	listInvites    *apporg.ListInvitationsUseCase
	revokeInvite   *apporg.RevokeInvitationUseCase
}

// NewOrganizationHandler creates an OrganizationHandler.
func NewOrganizationHandler(
	create *apporg.CreateOrganizationUseCase,
	get *apporg.GetOrganizationUseCase,
	update *apporg.UpdateOrganizationUseCase,
	delete *apporg.DeleteOrganizationUseCase,
	listMembers *apporg.ListMembersUseCase,
	updateMember *apporg.UpdateMemberUseCase,
	removeMember *apporg.RemoveMemberUseCase,
	invite *apporg.InviteMemberUseCase,
	acceptInvite *apporg.AcceptInvitationUseCase,
	listInvites *apporg.ListInvitationsUseCase,
	revokeInvite *apporg.RevokeInvitationUseCase,
) *OrganizationHandler {
	return &OrganizationHandler{
		create: create, get: get, update: update, delete: delete,
		listMembers: listMembers, updateMember: updateMember, removeMember: removeMember,
		invite: invite, acceptInvite: acceptInvite, listInvites: listInvites, revokeInvite: revokeInvite,
	}
}

type createOrganizationRequest struct {
	Name string `json:"name"`
}

type updateOrganizationRequest struct {
	Name string `json:"name"`
}

type organizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type memberResponse struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	JoinedAt string `json:"joined_at"`
}

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type inviteResponse struct {
	ID              string `json:"id"`
	InvitationToken string `json:"invitation_token"`
	ExpiresAt       string `json:"expires_at"`
}

type invitationResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

type updateMemberRequest struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

type acceptInvitationRequest struct {
	Token string `json:"token"`
}

// CreateOrganization handles POST /api/v1/organizations.
func (h *OrganizationHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	user, ok := httpmiddleware.AuthenticatedUserFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}
	var req createOrganizationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	org, err := h.create.Execute(r.Context(), apporg.CreateOrganizationInput{ActorUserID: user.UserID, Name: req.Name})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toOrganizationResponse(org))
}

// GetOrganization handles GET /api/v1/organizations/{organization_id}.
func (h *OrganizationHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization context required")
		return
	}
	orgID, _ := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
	org, err := h.get.Execute(r.Context(), apporg.GetOrganizationInput{OrganizationID: orgID, Principal: principal})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toOrganizationResponse(org))
}

// UpdateOrganization handles PATCH /api/v1/organizations/{organization_id}.
func (h *OrganizationHandler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization context required")
		return
	}
	orgID, _ := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
	var req updateOrganizationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	org, err := h.update.Execute(r.Context(), apporg.UpdateOrganizationInput{
		OrganizationID: orgID, Principal: principal, Name: req.Name,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toOrganizationResponse(org))
}

// DeleteOrganization handles DELETE /api/v1/organizations/{organization_id}.
func (h *OrganizationHandler) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization context required")
		return
	}
	orgID, _ := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
	if err := h.delete.Execute(r.Context(), apporg.DeleteOrganizationInput{OrganizationID: orgID, Principal: principal}); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListMembers handles GET /api/v1/organizations/{organization_id}/members.
func (h *OrganizationHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization context required")
		return
	}
	orgID, _ := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
	paging, err := parseOrgListQuery(r)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	members, cursor, err := h.listMembers.Execute(r.Context(), apporg.ListMembersInput{
		OrganizationID: orgID, Principal: principal, Limit: paging.Limit, Cursor: paging.Cursor,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	resp := make([]memberResponse, 0, len(members))
	for _, member := range members {
		resp = append(resp, memberResponse{
			ID: member.ID().String(), UserID: member.UserID().String(),
			Role: member.Role().String(), Status: string(member.Status()),
			JoinedAt: member.JoinedAt().Format(timeRFC3339),
		})
	}
	writeJSON(w, http.StatusOK, platformpagination.Collection[memberResponse]{
		Items: resp, NextCursor: cursor, HasMore: cursor != "",
	})
}

// UpdateMember handles PATCH /api/v1/organizations/{organization_id}/members/{user_id}.
func (h *OrganizationHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization context required")
		return
	}
	orgID, _ := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
	targetUserID, err := domain.ParseUserID(chi.URLParam(r, "user_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid user id")
		return
	}
	var req updateMemberRequest
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	member, err := h.updateMember.Execute(r.Context(), apporg.UpdateMemberInput{
		OrganizationID: orgID, Principal: principal, TargetUserID: targetUserID, Role: req.Role, Status: req.Status,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, memberResponse{
		ID: member.ID().String(), UserID: member.UserID().String(),
		Role: member.Role().String(), Status: string(member.Status()),
		JoinedAt: member.JoinedAt().Format(timeRFC3339),
	})
}

// RemoveMember handles DELETE /api/v1/organizations/{organization_id}/members/{user_id}.
func (h *OrganizationHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization context required")
		return
	}
	orgID, _ := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
	targetUserID, err := domain.ParseUserID(chi.URLParam(r, "user_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid user id")
		return
	}
	if err := h.removeMember.Execute(r.Context(), apporg.RemoveMemberInput{
		OrganizationID: orgID, Principal: principal, TargetUserID: targetUserID,
	}); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateInvitation handles POST /api/v1/organizations/{organization_id}/invitations.
func (h *OrganizationHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization context required")
		return
	}
	orgID, _ := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
	var req inviteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	result, err := h.invite.Execute(r.Context(), apporg.InviteMemberInput{
		OrganizationID: orgID, Principal: principal, Email: req.Email, Role: req.Role,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, inviteResponse{
		ID: result.InvitationID, InvitationToken: result.InvitationToken,
		ExpiresAt: result.ExpiresAt.Format(timeRFC3339),
	})
}

// ListInvitations handles GET /api/v1/organizations/{organization_id}/invitations.
func (h *OrganizationHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization context required")
		return
	}
	orgID, _ := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
	paging, err := parseOrgListQuery(r)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	invitations, cursor, err := h.listInvites.Execute(r.Context(), apporg.ListInvitationsInput{
		OrganizationID: orgID, Principal: principal, Limit: paging.Limit, Cursor: paging.Cursor,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	resp := make([]invitationResponse, 0, len(invitations))
	for _, inv := range invitations {
		resp = append(resp, invitationResponse{
			ID: inv.ID().String(), Email: inv.Email().String(), Role: inv.Role().String(),
			Status: string(inv.Status()), ExpiresAt: inv.ExpiresAt().Format(timeRFC3339),
			CreatedAt: inv.CreatedAt().Format(timeRFC3339),
		})
	}
	writeJSON(w, http.StatusOK, platformpagination.Collection[invitationResponse]{
		Items: resp, NextCursor: cursor, HasMore: cursor != "",
	})
}

// RevokeInvitation handles DELETE /api/v1/organizations/{organization_id}/invitations/{id}.
func (h *OrganizationHandler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization context required")
		return
	}
	orgID, _ := domain.ParseOrganizationID(chi.URLParam(r, "organization_id"))
	invID, err := domain.ParseInvitationID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid invitation id")
		return
	}
	if err := h.revokeInvite.Execute(r.Context(), apporg.RevokeInvitationInput{
		OrganizationID: orgID, Principal: principal, InvitationID: invID,
	}); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AcceptInvitation handles POST /api/v1/invitations/{token}/accept.
func (h *OrganizationHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	user, ok := httpmiddleware.AuthenticatedUserFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}
	token := chi.URLParam(r, "token")
	if token == "" {
		var req acceptInvitationRequest
		if err := decodeJSON(r, &req); err == nil && req.Token != "" {
			token = req.Token
		}
	}
	member, err := h.acceptInvite.Execute(r.Context(), apporg.AcceptInvitationInput{Actor: user, Token: token})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, memberResponse{
		ID: member.ID().String(), UserID: member.UserID().String(),
		Role: member.Role().String(), Status: string(member.Status()),
		JoinedAt: member.JoinedAt().Format(timeRFC3339),
	})
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func toOrganizationResponse(org *domain.Organization) organizationResponse {
	return organizationResponse{
		ID: org.ID().String(), Name: org.Name().String(), Slug: org.Slug().String(),
		Status: string(org.Status()), CreatedAt: org.CreatedAt().Format(timeRFC3339),
		UpdatedAt: org.UpdatedAt().Format(timeRFC3339),
	}
}
