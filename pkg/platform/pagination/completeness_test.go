package pagination_test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/stretchr/testify/require"
)

type pageRow struct {
	at time.Time
	id string
}

func TestPaginationCompleteness_NoDuplicatesOrGaps(t *testing.T) {
	t.Parallel()
	rows := make([]pageRow, 0, 120)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 120; i++ {
		rows = append(rows, pageRow{
			at: base.Add(time.Duration(i/10) * time.Second),
			id: fmt.Sprintf("row_%03d", i),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].at.Equal(rows[j].at) {
			return rows[i].id > rows[j].id
		}
		return rows[i].at.After(rows[j].at)
	})

	const pageSize = 17
	seen := make(map[string]struct{})
	cursor := ""
	pages := 0
	for {
		pages++
		batch := paginateRows(rows, cursor, pageSize)
		if len(batch) == 0 {
			break
		}
		for _, row := range batch {
			key := row.id
			if _, dup := seen[key]; dup {
				t.Fatalf("duplicate row %s on page %d", key, pages)
			}
			seen[key] = struct{}{}
		}
		if len(batch) < pageSize {
			break
		}
		last := batch[len(batch)-1]
		cursor = platformpagination.EncodeTimeIDCursor(last.at, last.id)
	}
	require.Equal(t, len(rows), len(seen), "missing rows after full pagination walk")
}

func TestPaginationCompleteness_DuplicateTimestamps(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	rows := []pageRow{
		{at: at, id: "c"},
		{at: at, id: "b"},
		{at: at, id: "a"},
		{at: at.Add(-time.Second), id: "z"},
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].at.Equal(rows[j].at) {
			return rows[i].id > rows[j].id
		}
		return rows[i].at.After(rows[j].at)
	})

	all := paginateRows(rows, "", 10)
	require.Len(t, all, 4)
	require.Equal(t, "c", all[0].id)
	require.Equal(t, "z", all[3].id)
}

func paginateRows(rows []pageRow, cursor string, limit int) []pageRow {
	if cursor == "" {
		end := limit
		if end > len(rows) {
			end = len(rows)
		}
		return rows[:end]
	}
	cursorTime, cursorID, err := platformpagination.DecodeTimeIDCursor(cursor)
	if err != nil {
		return nil
	}
	out := make([]pageRow, 0, limit)
	for _, row := range rows {
		if row.at.Before(cursorTime) || (row.at.Equal(cursorTime) && row.id < cursorID) {
			out = append(out, row)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}
