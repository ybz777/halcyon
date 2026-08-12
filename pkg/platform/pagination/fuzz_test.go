package pagination

import (
	"testing"
	"time"
)

func FuzzDecodeTimeIDCursor(f *testing.F) {
	f.Add("")
	f.Add("not-valid")
	f.Add(EncodeTimeIDCursor(mustTime(2026, 1, 1), "item_1"))
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 512 {
			t.Skip("bounded input")
		}
		_, _, _ = DecodeTimeIDCursor(raw)
	})
}

func mustTime(y int, m int, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}
