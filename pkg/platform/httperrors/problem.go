package httperrors

import (
	"encoding/json"
	"net/http"
)

// ProblemDetails implements RFC 7807 application/problem+json.
type ProblemDetails struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteProblem writes an RFC 7807 response.
func WriteProblem(w http.ResponseWriter, problem ProblemDetails) {
	if problem.Status == 0 {
		problem.Status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	_ = json.NewEncoder(w).Encode(problem)
}
