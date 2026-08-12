package jwks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Key represents a verification key from JWKS.
type Key struct {
	Kty string
	Kid string
	Alg string
	K   []byte
}

// Client fetches and caches JWKS verification keys.
type Client struct {
	url        string
	httpClient *http.Client
	cacheTTL   time.Duration
	mu         sync.RWMutex
	keys       map[string]Key
	fetchedAt  time.Time
	onRefresh  func(success bool)
}

// NewClient creates a JWKS client.
func NewClient(url string, cacheTTL time.Duration, httpClient *http.Client, onRefresh func(success bool)) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Client{
		url:        url,
		httpClient: httpClient,
		cacheTTL:   cacheTTL,
		keys:       make(map[string]Key),
		onRefresh:  onRefresh,
	}
}

type jwksResponse struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Alg string `json:"alg"`
		K   string `json:"k"`
	} `json:"keys"`
}

// KeyFunc returns a jwt keyfunc using cached JWKS keys.
func (c *Client) KeyFunc(ctx context.Context) jwt.Keyfunc {
	return func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Method.Alg())
		}
		kid, _ := t.Header["kid"].(string)
		key, err := c.getKey(ctx, kid)
		if err != nil {
			return nil, err
		}
		if key.Alg != "" && key.Alg != "HS256" {
			return nil, fmt.Errorf("unexpected key algorithm: %s", key.Alg)
		}
		return key.K, nil
	}
}

func (c *Client) getKey(ctx context.Context, kid string) (Key, error) {
	c.mu.RLock()
	stale := c.fetchedAt.IsZero() || time.Since(c.fetchedAt) > c.cacheTTL
	key, ok := c.keys[kid]
	c.mu.RUnlock()
	if ok && !stale {
		return key, nil
	}
	if err := c.refresh(ctx); err != nil {
		c.mu.RLock()
		defer c.mu.RUnlock()
		if key, ok := c.keys[kid]; ok {
			return key, nil
		}
		if len(c.keys) == 1 {
			for _, k := range c.keys {
				return k, nil
			}
		}
		return Key{}, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if key, ok := c.keys[kid]; ok {
		return key, nil
	}
	for _, k := range c.keys {
		return k, nil
	}
	return Key{}, fmt.Errorf("jwks key not found")
}

func (c *Client) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		c.recordRefresh(false)
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordRefresh(false)
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.recordRefresh(false)
		return fmt.Errorf("fetch jwks: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		c.recordRefresh(false)
		return err
	}
	var parsed jwksResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		c.recordRefresh(false)
		return err
	}
	keys := make(map[string]Key, len(parsed.Keys))
	for _, item := range parsed.Keys {
		if item.Kty != "oct" || item.K == "" {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(item.K)
		if err != nil {
			continue
		}
		keys[item.Kid] = Key{Kty: item.Kty, Kid: item.Kid, Alg: item.Alg, K: raw}
	}
	if len(keys) == 0 {
		c.recordRefresh(false)
		return fmt.Errorf("jwks contains no usable keys")
	}
	c.mu.Lock()
	c.keys = keys
	c.fetchedAt = time.Now().UTC()
	c.mu.Unlock()
	c.recordRefresh(true)
	return nil
}

func (c *Client) recordRefresh(success bool) {
	if c.onRefresh != nil {
		c.onRefresh(success)
	}
}

// Refresh forces a JWKS refresh.
func (c *Client) Refresh(ctx context.Context) error {
	return c.refresh(ctx)
}
