package apikey

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Principal struct {
	APIKeyID       string
	OrganizationID string
	Scopes         []string
}

// Authenticator validates API keys via the auth service introspection endpoint.
type Authenticator struct {
	client    *http.Client
	baseURL   string
	cacheTTL  time.Duration
	mu        sync.Mutex
	cache     map[string]cacheEntry
}

type cacheEntry struct {
	principal Principal
	expiresAt time.Time
}

func NewAuthenticator(baseURL string) *Authenticator {
	return &Authenticator{
		client:   &http.Client{Timeout: 5 * time.Second},
		baseURL:  strings.TrimRight(baseURL, "/"),
		cacheTTL: 60 * time.Second,
		cache:    make(map[string]cacheEntry),
	}
}

func (a *Authenticator) Authenticate(ctx context.Context, apiKey string) (Principal, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return Principal{}, fmt.Errorf("missing api key")
	}
	if principal, ok := a.fromCache(apiKey); ok {
		return principal, nil
	}

	body, _ := json.Marshal(map[string]string{"apiKey": apiKey})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/v1/auth/api-keys/introspect", bytes.NewReader(body))
	if err != nil {
		return Principal{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return Principal{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Principal{}, fmt.Errorf("api key authentication failed")
	}
	var payload struct {
		APIKeyID       string   `json:"api_key_id"`
		OrganizationID string   `json:"organization_id"`
		Scopes         []string `json:"scopes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Principal{}, err
	}
	principal := Principal{
		APIKeyID: payload.APIKeyID, OrganizationID: payload.OrganizationID, Scopes: payload.Scopes,
	}
	a.storeCache(apiKey, principal)
	return principal, nil
}

func (a *Authenticator) fromCache(apiKey string) (Principal, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, ok := a.cache[apiKey]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(a.cache, apiKey)
		return Principal{}, false
	}
	return entry.principal, true
}

func (a *Authenticator) storeCache(apiKey string, principal Principal) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache[apiKey] = cacheEntry{principal: principal, expiresAt: time.Now().Add(a.cacheTTL)}
}
