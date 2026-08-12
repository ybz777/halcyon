package pagination

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 200
)

// NormalizePageSize returns a safe page size when parsing is not required.
func NormalizePageSize(size, defaultSize, maxSize int) int {
	if defaultSize <= 0 {
		defaultSize = DefaultPageSize
	}
	if maxSize <= 0 {
		maxSize = MaxPageSize
	}
	if size <= 0 {
		return defaultSize
	}
	if size > maxSize {
		return maxSize
	}
	return size
}

// ParsePageSize validates a page size query value.
// Missing values use defaultSize. Zero, negative, non-numeric, and oversized values return ErrInvalidLimit.
func ParsePageSize(raw string, defaultSize, maxSize int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return NormalizePageSize(0, defaultSize, maxSize), nil
	}
	size, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: page size must be numeric", ErrInvalidLimit)
	}
	if size <= 0 {
		return 0, fmt.Errorf("%w: page size must be positive", ErrInvalidLimit)
	}
	if size > maxSize {
		return 0, fmt.Errorf("%w: page size exceeds maximum %d", ErrInvalidLimit, maxSize)
	}
	return size, nil
}
