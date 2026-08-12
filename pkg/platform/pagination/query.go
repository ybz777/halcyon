package pagination

import (
	"net/url"
	"strings"
	"time"
)

// Parser validates HTTP query parameters into application options.
type Parser struct {
	DefaultPageSize int
	MaxPageSize     int
}

// NewParser creates a query parser with page size defaults.
func NewParser(defaultSize, maxSize int) Parser {
	if defaultSize <= 0 {
		defaultSize = DefaultPageSize
	}
	if maxSize <= 0 {
		maxSize = MaxPageSize
	}
	return Parser{DefaultPageSize: defaultSize, MaxPageSize: maxSize}
}

// ParsePagination extracts cursor and page size from query values.
func (p Parser) ParsePagination(q url.Values) (PaginationOptions, error) {
	rawLimit := firstNonEmpty(q.Get("page_size"), q.Get("limit"))
	limit, err := ParsePageSize(rawLimit, p.DefaultPageSize, p.MaxPageSize)
	if err != nil {
		return PaginationOptions{}, err
	}
	return PaginationOptions{
		Limit:  limit,
		Cursor: strings.TrimSpace(q.Get("cursor")),
	}, nil
}

// ParseRFC3339Time parses an optional RFC3339 timestamp filter.
func ParseRFC3339Time(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, ErrInvalidFilter
	}
	return &t, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
