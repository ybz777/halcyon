package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const maxRequestIDLen = 128

// RequestID validates client-supplied request IDs and echoes the value.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if reqID == "" {
			reqID = generateRequestID()
		} else if len(reqID) > maxRequestIDLen {
			reqID = reqID[:maxRequestIDLen]
		}
		r.Header.Set("X-Request-ID", reqID)
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

func generateRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "req_" + hex.EncodeToString(b[:])
}
