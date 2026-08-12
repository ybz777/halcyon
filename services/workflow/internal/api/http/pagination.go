package httpapi

import (
	"net/http"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
)

var workflowQueryParser = platformpagination.NewParser(20, 100)

func parseListQuery(r *http.Request) (platformpagination.PaginationOptions, error) {
	return workflowQueryParser.ParsePagination(r.URL.Query())
}

func collectionResponse[T any](items []T, nextCursor string) platformpagination.Collection[T] {
	if items == nil {
		items = []T{}
	}
	return platformpagination.Collection[T]{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}
}
