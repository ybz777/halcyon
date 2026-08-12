package ratelimit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// IdentityType classifies a rate-limit subject.
type IdentityType string

const (
	IdentityIP     IdentityType = "ip"
	IdentityUser   IdentityType = "user"
	IdentityOrg    IdentityType = "org"
	IdentityAPIKey IdentityType = "apikey"
)

// BuildKey creates a bounded Redis key component from identity parts.
func BuildKey(parts ...string) string {
	return strings.Join(parts, ":")
}

// RedisKey returns the full namespaced Redis key.
func RedisKey(prefix, policy, identity string) string {
	hashed := hashIdentity(identity)
	if len(hashed) > 64 {
		hashed = hashed[:64]
	}
	return fmt.Sprintf("%s:%s:%s", sanitize(prefix), sanitize(policy), hashed)
}

func hashIdentity(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:16])
}

func sanitize(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, " ", "_")
	if len(v) > 64 {
		v = v[:64]
	}
	return v
}

// ClientIP resolves the client IP with optional trusted proxy support.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := NormalizeIP(strings.TrimSpace(parts[0])); ip != "" {
				return ip
			}
		}
		if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
			if ip := NormalizeIP(xri); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return NormalizeIP(r.RemoteAddr)
	}
	return NormalizeIP(host)
}

// NormalizeIP canonicalizes IPv4 and IPv6 addresses.
func NormalizeIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return raw
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.String()
}

// IdentityKey builds a stable identity string.
func IdentityKey(kind IdentityType, value string) string {
	return BuildKey(string(kind), value)
}
