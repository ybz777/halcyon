package domain



import (

	"time"



	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"

)



// ListFilter scopes audit queries.

type ListFilter struct {

	OrganizationID string

	EventType      string

	ActorID        string

	ActorType      ActorType

	TargetType     TargetType

	TargetID       string

	Result         Result

	StartTime      *time.Time

	EndTime        *time.Time

	PageSize       int

	Cursor         string

	IncludeSecurity bool

}



const (

	DefaultPageSize = platformpagination.DefaultPageSize

	MaxPageSize     = platformpagination.MaxPageSize

)



// NormalizePageSize clamps page size to safe bounds.

func NormalizePageSize(size int) int {

	return platformpagination.NormalizePageSize(size, DefaultPageSize, MaxPageSize)

}



// EncodeCursor creates an opaque pagination cursor.

func EncodeCursor(occurredAt time.Time, id string) string {

	return platformpagination.EncodeTimeIDCursor(occurredAt, id)

}



// DecodeCursor parses an opaque cursor.

func DecodeCursor(raw string) (time.Time, string, error) {

	at, id, err := platformpagination.DecodeTimeIDCursor(raw)

	if err != nil {

		return time.Time{}, "", ErrInvalidCursor

	}

	return at, id, nil

}


