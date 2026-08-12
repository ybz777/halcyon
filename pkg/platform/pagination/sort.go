package pagination

import (
	"fmt"
	"strings"
)

// Direction is a validated sort direction.
type Direction string

const (
	Asc  Direction = "asc"
	Desc Direction = "desc"
)

// Sort describes validated sort options.
type Sort struct {
	Field     string
	Direction Direction
	SQLColumn string
}

// ParseDirection validates a sort direction.
func ParseDirection(raw string) (Direction, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "desc":
		return Desc, nil
	case "asc":
		return Asc, nil
	default:
		return "", fmt.Errorf("%w: invalid sort direction", ErrInvalidSort)
	}
}

// ParseSort validates a sort field against an allowlist mapping API names to SQL columns.
func ParseSort(field, direction string, allowed map[string]string, defaultField string) (Sort, error) {
	field = strings.TrimSpace(field)
	if field == "" {
		field = defaultField
	}
	sqlColumn, ok := allowed[field]
	if !ok {
		return Sort{}, fmt.Errorf("%w: unsupported sort field %q", ErrInvalidSort, field)
	}
	dir, err := ParseDirection(direction)
	if err != nil {
		return Sort{}, err
	}
	return Sort{Field: field, Direction: dir, SQLColumn: sqlColumn}, nil
}
