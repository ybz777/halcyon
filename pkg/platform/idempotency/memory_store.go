package idempotency

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type captureWriter struct {
	http.ResponseWriter
	status      int
	body        []byte
	maxBytes    int64
	captureBody bool
	header      http.Header
	wroteHeader bool
}

func newCaptureWriter(w http.ResponseWriter, maxBytes int64, captureBody bool) *captureWriter {
	return &captureWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
		maxBytes:       maxBytes,
		captureBody:    captureBody,
		header:         make(http.Header),
	}
}

func (w *captureWriter) Header() http.Header {
	return w.header
}

func (w *captureWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
}

func (w *captureWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.captureBody && int64(len(w.body)) < w.maxBytes {
		remaining := w.maxBytes - int64(len(w.body))
		if int64(len(b)) <= remaining {
			w.body = append(w.body, b...)
		} else if remaining > 0 {
			w.body = append(w.body, b[:remaining]...)
		}
	}
	return len(b), nil
}

func (w *captureWriter) flush() {
	dst := w.ResponseWriter
	for k, vals := range w.header {
		if isSensitiveHeader(k) {
			continue
		}
		for _, v := range vals {
			dst.Header().Add(k, v)
		}
	}
	status := w.status
	if !w.wroteHeader {
		status = http.StatusOK
	}
	dst.WriteHeader(status)
	if w.captureBody {
		_, _ = dst.Write(w.body)
	}
}

func isSensitiveHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "set-cookie", "authorization", "www-authenticate", "proxy-authenticate":
		return true
	default:
		return false
	}
}

func replayResponse(w http.ResponseWriter, record Record) {
	w.Header().Set(HeaderReplay, "true")
	if ct := strings.TrimSpace(record.ContentType); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	status := record.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if len(record.Body) > 0 {
		_, _ = w.Write(record.Body)
	}
}

func shouldCacheResponse(status int) bool {
	if status >= 200 && status < 500 {
		return status < 500
	}
	return status >= 400 && status < 500
}

func isTransientStatus(status int) bool {
	return status >= 500
}

// memoryStore is an in-memory Store for unit tests.
type memoryStore struct {
	mu      sync.Mutex
	records map[string]Record
}

func NewMemoryStore() *memoryStore {
	return &memoryStore{records: make(map[string]Record)}
}

func (s *memoryStore) Claim(_ context.Context, key, fingerprint string, _ time.Duration) (ClaimOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.records[key]
	if !ok {
		rec := Record{State: StateInProgress, Fingerprint: fingerprint, CreatedAt: time.Now().UTC()}
		s.records[key] = rec
		return ClaimOutcome{Result: ClaimAcquired, Record: rec}, nil
	}
	if existing.Fingerprint != fingerprint {
		return ClaimOutcome{Result: ClaimConflict, Record: existing}, nil
	}
	switch existing.State {
	case StateCompleted:
		return ClaimOutcome{Result: ClaimReplay, Record: existing}, nil
	default:
		return ClaimOutcome{Result: ClaimInProgress, Record: existing}, nil
	}
}

func (s *memoryStore) Complete(_ context.Context, key, fingerprint string, record Record, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.records[key]
	if ok && existing.Fingerprint != fingerprint {
		return ErrFingerprintMismatch
	}
	record.State = StateCompleted
	record.Fingerprint = fingerprint
	record.CompletedAt = time.Now().UTC()
	s.records[key] = record
	return nil
}

func (s *memoryStore) Release(_ context.Context, key, fingerprint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.records[key]
	if !ok {
		return nil
	}
	if existing.State == StateInProgress && existing.Fingerprint == fingerprint {
		delete(s.records, key)
	}
	return nil
}

func encodeRecord(rec Record) ([]byte, error) {
	return json.Marshal(rec)
}

func decodeRecord(data []byte) (Record, error) {
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, err
	}
	return rec, nil
}
