package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const cursorVersion = 1

type timeIDPayload struct {
	Version int       `json:"v"`
	Time    time.Time `json:"t"`
	ID      string    `json:"id"`
}

type keysetPayload struct {
	Version   int    `json:"v"`
	SortField string `json:"sf,omitempty"`
	SortDir   string `json:"sd,omitempty"`
	Primary   string `json:"p"`
	ID        string `json:"id"`
}

// EncodeTimeIDCursor creates an opaque cursor for time+id keyset pagination.
func EncodeTimeIDCursor(at time.Time, id string) string {
	payload, _ := json.Marshal(timeIDPayload{Version: cursorVersion, Time: at.UTC(), ID: id})
	return base64.RawURLEncoding.EncodeToString(payload)
}

// DecodeTimeIDCursor parses a time+id cursor.
func DecodeTimeIDCursor(raw string) (time.Time, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, "", nil
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("%w: decode cursor", ErrInvalidCursor)
	}
	var payload timeIDPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return time.Time{}, "", fmt.Errorf("%w: parse cursor", ErrInvalidCursor)
	}
	if payload.Version != cursorVersion || payload.ID == "" || payload.Time.IsZero() {
		return time.Time{}, "", ErrInvalidCursor
	}
	return payload.Time, payload.ID, nil
}

// EncodeKeysetCursor creates an opaque cursor for dynamic sort keyset pagination.
func EncodeKeysetCursor(sortField, sortDir, primary, id string) string {
	payload, _ := json.Marshal(keysetPayload{
		Version:   cursorVersion,
		SortField: sortField,
		SortDir:   sortDir,
		Primary:   primary,
		ID:        id,
	})
	return base64.RawURLEncoding.EncodeToString(payload)
}

// KeysetCursor represents a decoded keyset cursor.
type KeysetCursor struct {
	SortField string
	SortDir   Direction
	Primary   string
	ID        string
}

// DecodeKeysetCursor parses a dynamic keyset cursor.
func DecodeKeysetCursor(raw string) (KeysetCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return KeysetCursor{}, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return KeysetCursor{}, fmt.Errorf("%w: decode cursor", ErrInvalidCursor)
	}
	var payload keysetPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return KeysetCursor{}, fmt.Errorf("%w: parse cursor", ErrInvalidCursor)
	}
	if payload.Version != cursorVersion || payload.ID == "" || payload.Primary == "" {
		return KeysetCursor{}, ErrInvalidCursor
	}
	dir, err := ParseDirection(payload.SortDir)
	if err != nil && payload.SortDir != "" {
		return KeysetCursor{}, err
	}
	if dir == "" {
		dir = Desc
	}
	return KeysetCursor{
		SortField: payload.SortField,
		SortDir:   dir,
		Primary:   payload.Primary,
		ID:        payload.ID,
	}, nil
}
