package kafka_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	kafkago "github.com/segmentio/kafka-go"
)

func TestDecodeKafkaMessageHeaders(t *testing.T) {
	t.Parallel()

	kmsg := kafkago.Message{
		Topic:     "halcyon.auth",
		Key:       []byte("org_1"),
		Value:     []byte(`{}`),
		Partition: 2,
		Offset:    99,
		Headers: []kafkago.Header{
			{Key: "event_id", Value: []byte("evt_test12345678901")},
			{Key: "event_type", Value: []byte("auth.user.registered")},
			{Key: "event_version", Value: []byte("1.0")},
			{Key: "correlation_id", Value: []byte("corr_1")},
			{Key: "request_id", Value: []byte("req_1")},
			{Key: "producer", Value: []byte("auth")},
		},
	}

	msg := kafka.DecodeMessage(kmsg)
	if msg.EventID != "evt_test12345678901" {
		t.Fatalf("event_id = %q", msg.EventID)
	}
	if msg.EventType != "auth.user.registered" {
		t.Fatalf("event_type = %q", msg.EventType)
	}
	if msg.EventVersion != "1.0" {
		t.Fatalf("event_version = %q", msg.EventVersion)
	}
	if msg.CorrelationID != "corr_1" || msg.RequestID != "req_1" || msg.Producer != "auth" {
		t.Fatalf("headers mismatch: %+v", msg)
	}
	if msg.Partition != 2 || msg.Offset != 99 {
		t.Fatalf("partition/offset = %d/%d", msg.Partition, msg.Offset)
	}
}
