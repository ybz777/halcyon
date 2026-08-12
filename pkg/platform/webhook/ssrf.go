package webhook

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// URLPolicy controls webhook destination validation.
type URLPolicy struct {
	AllowHTTP bool
}

// ValidateDestinationURL rejects SSRF-prone webhook targets.
func ValidateDestinationURL(raw string, policy URLPolicy) error {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "https" {
		if !(policy.AllowHTTP && u.Scheme == "http") {
			return fmt.Errorf("only https destinations are allowed")
		}
	}
	if u.User != nil {
		return fmt.Errorf("userinfo in url is not allowed")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("host is required")
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return fmt.Errorf("localhost destinations are not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("private or reserved ip destinations are not allowed")
		}
	}
	if isMetadataHost(lower) {
		return fmt.Errorf("metadata endpoints are not allowed")
	}
	return nil
}

// ResolveAndValidate resolves DNS and ensures all addresses are public.
func ResolveAndValidate(raw string, policy URLPolicy) error {
	if err := ValidateDestinationURL(raw, policy); err != nil {
		return err
	}
	u, _ := url.Parse(raw)
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("private ip destination")
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("dns lookup failed: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("dns lookup returned no addresses")
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("destination resolves to blocked ip")
		}
	}
	return nil
}

func isMetadataHost(host string) bool {
	switch host {
	case "metadata.google.internal", "169.254.169.254":
		return true
	default:
		return false
	}
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	privateRanges := []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"127.0.0.0/8", "0.0.0.0/8",
		"fc00::/7", "::1/128", "fe80::/10",
	}
	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
