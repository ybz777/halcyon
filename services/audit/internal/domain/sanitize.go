package domain

import (
	"strings"
)

var sensitiveKeys = map[string]struct{}{
	"password": {}, "passwd": {}, "secret": {}, "token": {}, "jwt": {},
	"refresh_token": {}, "access_token": {}, "authorization": {}, "cookie": {},
	"api_key": {}, "private_key": {}, "credential": {}, "session": {},
}

// SanitizeMetadata removes sensitive values from audit metadata.
func SanitizeMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		key := strings.ToLower(strings.TrimSpace(k))
		if _, blocked := sensitiveKeys[key]; blocked {
			continue
		}
		if strings.Contains(strings.ToLower(key), "password") ||
			strings.Contains(strings.ToLower(key), "token") ||
			strings.Contains(strings.ToLower(key), "secret") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(v), "bearer ") {
			continue
		}
		out[k] = v
	}
	return out
}

// SanitizeString rejects values that look like secrets.
func SanitizeString(value string) string {
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "bearer ") || strings.HasPrefix(lower, "eyj") {
		return ""
	}
	return value
}
