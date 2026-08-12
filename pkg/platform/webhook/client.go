package webhook

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// ClientConfig configures outbound webhook HTTP delivery.
type ClientConfig struct {
	Timeout             time.Duration
	ResponseBodyMaxSize int
	AllowHTTP           bool
}

// DefaultClientConfig returns safe defaults.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Timeout:             10 * time.Second,
		ResponseBodyMaxSize: 1024,
	}
}

// DeliveryResult captures bounded response metadata.
type DeliveryResult struct {
	StatusCode     int
	ResponseTimeMs int64
	ErrorSummary   string
	Retryable      bool
}

// Client delivers signed webhook HTTP requests.
type Client struct {
	httpClient *http.Client
	cfg        ClientConfig
}

// NewClient creates a shared webhook HTTP client.
func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout <= 0 {
		cfg = DefaultClientConfig()
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if ip := net.ParseIP(host); ip != nil && isBlockedIP(ip) {
				return nil, fmt.Errorf("blocked destination ip")
			}
			dialer := &net.Dialer{Timeout: 5 * time.Second}
			return dialer.DialContext(ctx, network, addr)
		},
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: cfg.Timeout,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
	}
	return &Client{
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		cfg: cfg,
	}
}

// Deliver sends a signed POST to the destination URL.
func (c *Client) Deliver(ctx context.Context, destinationURL, secret, eventID, eventType string, payload []byte) DeliveryResult {
	start := time.Now()
	if err := ResolveAndValidate(destinationURL, URLPolicy{AllowHTTP: c.cfg.AllowHTTP}); err != nil {
		return DeliveryResult{ErrorSummary: truncate(err.Error(), 200), Retryable: false}
	}

	ts := time.Now().UTC().Unix()
	signature := SignPayload(secret, ts, payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, destinationURL, strings.NewReader(string(payload)))
	if err != nil {
		return DeliveryResult{ErrorSummary: truncate(err.Error(), 200), Retryable: false}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(TimestampHeader, fmt.Sprintf("%d", ts))
	req.Header.Set(SignatureHeader, signature)
	req.Header.Set(EventIDHeader, eventID)
	req.Header.Set(EventTypeHeader, eventType)

	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return DeliveryResult{ResponseTimeMs: elapsed, ErrorSummary: truncate(err.Error(), 200), Retryable: true}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, int64(c.cfg.ResponseBodyMaxSize)))

	result := DeliveryResult{
		StatusCode:     resp.StatusCode,
		ResponseTimeMs: elapsed,
		Retryable:      classifyRetry(resp.StatusCode),
	}
	if resp.StatusCode >= 400 {
		result.ErrorSummary = truncate(http.StatusText(resp.StatusCode), 200)
	}
	return result
}

func classifyRetry(status int) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	return status >= 500
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
