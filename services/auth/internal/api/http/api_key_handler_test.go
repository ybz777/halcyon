package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

func TestWriteAPIKeyError_ServiceAccountNotFoundIsBadRequest(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", nil)

	writeAPIKeyError(rec, req, domain.ErrInvalidServiceAccount)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestWriteAPIKeyError_ForbiddenIs403(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", nil)

	writeAPIKeyError(rec, req, domain.ErrForbidden)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestWriteAPIKeyError_UnknownIs500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", nil)

	writeAPIKeyError(rec, req, errors.New("unexpected"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
