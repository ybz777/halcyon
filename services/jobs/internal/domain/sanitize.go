package domain

import (
	"encoding/json"
	"regexp"
	"strings"
)

var secretFieldPattern = regexp.MustCompile(`(?i)(token|secret|password|api[_-]?key|authorization|credential)`)

// PayloadSummary returns a sanitized view of a job payload for admin APIs.
func PayloadSummary(payload []byte) map[string]any {
	if len(payload) == 0 {
		return map[string]any{}
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return map[string]any{"summary": "unparseable payload"}
	}
	out := make(map[string]any, len(raw))
	for key, value := range raw {
		if secretFieldPattern.MatchString(key) {
			out[key] = "[redacted]"
			continue
		}
		switch typed := value.(type) {
		case string:
			if len(typed) > 256 {
				out[key] = typed[:256] + "..."
			} else {
				out[key] = typed
			}
		default:
			out[key] = value
		}
	}
	return out
}

// NormalizePageSize clamps list page size.
func NormalizePageSize(size int) int {
	if size <= 0 {
		return 50
	}
	if size > 200 {
		return 200
	}
	return size
}

// TrustedOrganization returns the organization ID from internal gateway headers.
func TrustedOrganization(header string) string {
	return strings.TrimSpace(header)
}
