package httputil

import (
	"encoding/json"
	"net/http"
)

// WriteJSON encodes body as JSON with the given HTTP status code.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
