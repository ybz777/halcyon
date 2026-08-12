package httperrors_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/stretchr/testify/require"
)

func TestWriteProblem_RFC7807Contract(t *testing.T) {
	t.Parallel()
	statuses := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	}
	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			rec := httptest.NewRecorder()
			httperrors.WriteProblem(rec, httperrors.ProblemDetails{
				Title:  http.StatusText(status),
				Status: status,
				Detail: "test detail",
			})
			require.Equal(t, status, rec.Code)
			require.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
			var body httperrors.ProblemDetails
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, status, body.Status)
			require.Equal(t, http.StatusText(status), body.Title)
		})
	}
}

func TestWriteProblem_DoesNotLeakSecrets(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	secret := "super-secret-token-value"
	httperrors.WriteProblem(rec, httperrors.ProblemDetails{
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: "authentication required",
	})
	require.NotContains(t, rec.Body.String(), secret)
}
