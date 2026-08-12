package pagination

// Options contains validated list query options for application layers.
type Options struct {
	Limit  int
	Cursor string
	Sort   Sort
}

// PaginationOptions contains validated pagination parameters.
type PaginationOptions struct {
	Limit  int
	Cursor string
}
