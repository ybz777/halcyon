package pagination

import "errors"

var (
	ErrInvalidCursor = errors.New("invalid cursor")
	ErrInvalidLimit  = errors.New("invalid page size")
	ErrInvalidSort   = errors.New("invalid sort")
	ErrInvalidFilter = errors.New("invalid filter")
)
