package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
)

var errMalformedJSON = errors.New("malformed json")

func decodeJSONBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return errMalformedJSON
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return errMalformedJSON
	}
	return nil
}

func validateRequired(fields map[string]string) []httperrors.FieldError {
	var errs []httperrors.FieldError
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, httperrors.FieldError{
				Field:   field,
				Code:    "required",
				Message: field + " is required",
			})
		}
	}
	return errs
}

func writeValidationProblem(w http.ResponseWriter, r *http.Request, detail string, fieldErrors []httperrors.FieldError) {
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Type:     "https://halcyon.io/problems/validation-error",
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Detail:   detail,
		Instance: middleware.GetReqID(r.Context()),
		Errors:   fieldErrors,
	})
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}
