package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// FingerprintInput captures request properties for idempotency binding.
type FingerprintInput struct {
	Method         string
	RoutePattern   string
	PrincipalType  string
	PrincipalID    string
	OrganizationID string
	Query          string
	Body           []byte
}

// Fingerprint computes a deterministic SHA-256 fingerprint for a request.
func Fingerprint(in FingerprintInput) (string, error) {
	canonicalBody, err := canonicalJSON(in.Body)
	if err != nil {
		return "", err
	}
	parts := []string{
		strings.ToUpper(strings.TrimSpace(in.Method)),
		strings.TrimSpace(in.RoutePattern),
		strings.TrimSpace(in.PrincipalType),
		strings.TrimSpace(in.PrincipalID),
		strings.TrimSpace(in.OrganizationID),
		strings.TrimSpace(in.Query),
		canonicalBody,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

// ReadBody reads and restores the request body up to maxBytes.
func ReadBody(r *http.Request, maxBytes int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("request body exceeds maximum allowed size")
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))
	return body, nil
}

func canonicalJSON(body []byte) (string, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return "", nil
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return string(body), nil
	}
	normalized, err := json.Marshal(sortJSON(v))
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}

func sortJSON(v any) any {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(t))
		for _, k := range keys {
			out[k] = sortJSON(t[k])
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = sortJSON(item)
		}
		return out
	default:
		return v
	}
}
