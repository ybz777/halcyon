package httpapi

import (
	"net/http"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
)

var orgQueryParser = platformpagination.NewParser(platformpagination.DefaultPageSize, platformpagination.MaxPageSize)

func parseOrgListQuery(r *http.Request) (platformpagination.PaginationOptions, error) {
	return orgQueryParser.ParsePagination(r.URL.Query())
}
