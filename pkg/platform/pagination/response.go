package pagination

// Collection is the standard list response envelope.
type Collection[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// BuildCollection trims a limit+1 result set into a page with cursor metadata.
func BuildCollection[T any](items []T, limit int, cursorFn func(T) string) Collection[T] {
	if len(items) == 0 {
		return Collection[T]{Items: []T{}, HasMore: false}
	}
	hasMore := len(items) > limit
	page := items
	if hasMore {
		page = items[:limit]
	}
	var next string
	if hasMore && cursorFn != nil {
		next = cursorFn(page[len(page)-1])
	}
	return Collection[T]{Items: page, NextCursor: next, HasMore: hasMore}
}
