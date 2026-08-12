package pagination

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testRow struct {
	at time.Time
	id string
}

func filterRowsAfterCursor(rows []testRow, cursor string, max int) []testRow {
	if cursor == "" {
		end := max
		if end > len(rows) {
			end = len(rows)
		}
		return rows[:end]
	}
	cursorTime, cursorID, err := DecodeTimeIDCursor(cursor)
	if err != nil {
		return nil
	}
	out := make([]testRow, 0, max)
	for _, row := range rows {
		if row.at.Before(cursorTime) || (row.at.Equal(cursorTime) && row.id < cursorID) {
			out = append(out, row)
			if len(out) >= max {
				break
			}
		}
	}
	return out
}

func TestParsePageSize(t *testing.T) {
	size, err := ParsePageSize("", 50, 200)
	require.NoError(t, err)
	require.Equal(t, 50, size)

	_, err = ParsePageSize("0", 50, 200)
	require.ErrorIs(t, err, ErrInvalidLimit)

	_, err = ParsePageSize("-1", 50, 200)
	require.ErrorIs(t, err, ErrInvalidLimit)

	_, err = ParsePageSize("abc", 50, 200)
	require.ErrorIs(t, err, ErrInvalidLimit)

	_, err = ParsePageSize("1000000", 50, 200)
	require.ErrorIs(t, err, ErrInvalidLimit)

	size, err = ParsePageSize("25", 50, 200)
	require.NoError(t, err)
	require.Equal(t, 25, size)
}

func TestTimeIDCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	encoded := EncodeTimeIDCursor(at, "item_123")
	gotAt, gotID, err := DecodeTimeIDCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, at, gotAt)
	require.Equal(t, "item_123", gotID)
}

func TestTimeIDCursorTamperRejected(t *testing.T) {
	_, _, err := DecodeTimeIDCursor("not-a-cursor")
	require.ErrorIs(t, err, ErrInvalidCursor)

	encoded := EncodeTimeIDCursor(time.Now().UTC(), "")
	_, _, err = DecodeTimeIDCursor(encoded)
	require.ErrorIs(t, err, ErrInvalidCursor)
}

func TestBuildCollection_HasMore(t *testing.T) {
	items := []string{"a", "b", "c"}
	page := BuildCollection(items, 2, func(v string) string { return v })
	require.True(t, page.HasMore)
	require.Equal(t, "b", page.NextCursor)
	require.Len(t, page.Items, 2)

	page = BuildCollection([]string{}, 10, nil)
	require.False(t, page.HasMore)
	require.NotNil(t, page.Items)
	require.Len(t, page.Items, 0)
}

func TestParseSortAllowlist(t *testing.T) {
	allowed := map[string]string{"created_at": "created_at", "status": "status"}
	sort, err := ParseSort("created_at", "desc", allowed, "created_at")
	require.NoError(t, err)
	require.Equal(t, Desc, sort.Direction)

	_, err = ParseSort("password", "asc", allowed, "created_at")
	require.ErrorIs(t, err, ErrInvalidSort)

	_, err = ParseSort("created_at", "sideways", allowed, "created_at")
	require.ErrorIs(t, err, ErrInvalidSort)
}

func TestPageWalkInvariant(t *testing.T) {
	rows := make([]testRow, 0, 25)
	for i := 0; i < 25; i++ {
		rows = append(rows, testRow{at: time.Unix(int64(25-i), 0).UTC(), id: string(rune('a' + i))})
	}
	limit := 5
	seen := map[string]struct{}{}
	cursor := ""
	for page := 0; page < 20; page++ {
		batch := filterRowsAfterCursor(rows, cursor, limit+1)
		pageResult := BuildCollection(batch, limit, func(r testRow) string { return EncodeTimeIDCursor(r.at, r.id) })
		for _, item := range pageResult.Items {
			seen[item.id] = struct{}{}
		}
		if !pageResult.HasMore {
			break
		}
		cursor = pageResult.NextCursor
	}
	require.Len(t, seen, 25)
}

func TestConcurrentCursorEncode(t *testing.T) {
	const workers = 32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			at := time.Unix(int64(i), 0).UTC()
			encoded := EncodeTimeIDCursor(at, "id")
			_, _, err := DecodeTimeIDCursor(encoded)
			require.NoError(t, err)
		}(i)
	}
	wg.Wait()
}
