package pagination

import (
	"testing"
	"time"
)

func BenchmarkEncodeTimeIDCursor(b *testing.B) {
	at := benchTime()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = EncodeTimeIDCursor(at, "aud_test123")
	}
}

func BenchmarkDecodeTimeIDCursor(b *testing.B) {
	raw := EncodeTimeIDCursor(benchTime(), "aud_test123")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeTimeIDCursor(raw)
	}
}

func BenchmarkParsePageSize(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParsePageSize("50", 50, 200)
	}
}

func benchTime() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}
