package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	appnotification "github.com/halcyon-io/halcyon/services/notifications/internal/application/notification"
	apppreference "github.com/halcyon-io/halcyon/services/notifications/internal/application/preference"
	httpmiddleware "github.com/halcyon-io/halcyon/services/notifications/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

type NotificationHandler struct {
	service *appnotification.Service
}

func NewNotificationHandler(service *appnotification.Service) *NotificationHandler {
	return &NotificationHandler{service: service}
}

var notificationQueryParser = platformpagination.NewParser(domain.DefaultPageSize, domain.MaxPageSize)

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := selfContext(w, r)
	if !ok {
		return
	}
	paging, err := notificationQueryParser.ParsePagination(r.URL.Query())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	items, cursor, err := h.service.List(r.Context(), orgID, userID, paging.Limit, paging.Cursor)
	if err != nil {
		writeError(w, err)
		return
	}
	if items == nil {
		items = []domain.Notification{}
	}
	page := platformpagination.Collection[map[string]any]{
		Items: toResponses(items), NextCursor: cursor, HasMore: cursor != "",
	}
	httputil.WriteJSON(w, http.StatusOK, page)
}

func (h *NotificationHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := selfContext(w, r)
	if !ok {
		return
	}
	id, err := domain.ParseNotificationID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	item, err := h.service.Get(r.Context(), orgID, userID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(item))
}

func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := selfContext(w, r)
	if !ok {
		return
	}
	count, err := h.service.UnreadCount(r.Context(), orgID, userID)
	if err != nil {
		writeError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"unread_count": count})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := selfContext(w, r)
	if !ok {
		return
	}
	id, err := domain.ParseNotificationID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := h.service.MarkRead(r.Context(), orgID, userID, id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := selfContext(w, r)
	if !ok {
		return
	}
	count, err := h.service.MarkAllRead(r.Context(), orgID, userID)
	if err != nil {
		writeError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"marked_read": count})
}

type PreferenceHandler struct {
	service *apppreference.Service
}

func NewPreferenceHandler(service *apppreference.Service) *PreferenceHandler {
	return &PreferenceHandler{service: service}
}

func (h *PreferenceHandler) List(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := selfContext(w, r)
	if !ok {
		return
	}
	paging, err := notificationQueryParser.ParsePagination(r.URL.Query())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	items, cursor, err := h.service.List(r.Context(), userID, paging.Limit, paging.Cursor)
	if err != nil {
		writeError(w, err)
		return
	}
	if items == nil {
		items = []domain.NotificationPreference{}
	}
	httputil.WriteJSON(w, http.StatusOK, platformpagination.Collection[map[string]any]{
		Items: toPreferenceResponses(items), NextCursor: cursor, HasMore: cursor != "",
	})
}

func (h *PreferenceHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := selfContext(w, r)
	if !ok {
		return
	}
	notificationType := domain.NotificationType(chi.URLParam(r, "notification_type"))
	var req struct {
		Channel string `json:"channel"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}
	pref, err := h.service.Update(r.Context(), userID, notificationType, domain.Channel(req.Channel), req.Enabled)
	if err != nil {
		writeError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toPreferenceResponse(pref))
}

func selfContext(w http.ResponseWriter, r *http.Request) (orgID, userID string, ok bool) {
	principal, found := httpmiddleware.PrincipalFromContext(r.Context())
	if !found || principal.UserID == "" {
		writeProblem(w, http.StatusUnauthorized, "authentication required")
		return "", "", false
	}
	orgID = httpmiddleware.OrganizationFromRequest(r)
	if orgID == "" {
		writeProblem(w, http.StatusBadRequest, "organization context required")
		return "", "", false
	}
	return orgID, principal.UserID, true
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: http.StatusText(status), Status: status, Detail: detail})
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotificationNotFound):
		writeProblem(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrInvalidPreference):
		writeProblem(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInvalidCursor), errors.Is(err, platformpagination.ErrInvalidLimit):
		writeProblem(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeProblem(w, http.StatusForbidden, "forbidden")
	default:
		writeProblem(w, http.StatusInternalServerError, "unexpected error")
	}
}

func toResponses(items []domain.Notification) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, toResponse(item))
	}
	return out
}

func toResponse(item domain.Notification) map[string]any {
	resp := map[string]any{
		"id": item.ID.String(), "organization_id": item.OrganizationID, "user_id": item.UserID,
		"event_type": item.EventType, "notification_type": item.NotificationType,
		"title": item.Title, "body": item.Body, "read": !item.IsUnread(), "created_at": item.CreatedAt,
	}
	if item.ReadAt != nil {
		resp["read_at"] = item.ReadAt
	}
	return resp
}

func toPreferenceResponses(items []domain.NotificationPreference) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, toPreferenceResponse(item))
	}
	return out
}

func toPreferenceResponse(item domain.NotificationPreference) map[string]any {
	return map[string]any{
		"notification_type": item.NotificationType, "channel": item.Channel,
		"enabled": item.Enabled, "updated_at": item.UpdatedAt,
	}
}
