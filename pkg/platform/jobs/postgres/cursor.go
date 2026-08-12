package postgres

import (
	"fmt"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
)

func encodeCursor(createdAt time.Time, id string) string {
	return platformpagination.EncodeTimeIDCursor(createdAt, id)
}

func decodeCursor(raw string) (time.Time, string, error) {
	at, id, err := platformpagination.DecodeTimeIDCursor(raw)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	if id != "" {
		if _, err := platformjobs.ParseID(id); err != nil {
			return time.Time{}, "", fmt.Errorf("invalid cursor")
		}
	}
	return at, id, nil
}
