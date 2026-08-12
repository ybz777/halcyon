package events

import (
	"encoding/json"
	"time"
)

// DLQMessage wraps a failed event with replay metadata.
type DLQMessage struct {
	OriginalEvent   EventEnvelope `json:"original_event"`
	OriginalTopic   string        `json:"original_topic"`
	Partition       int           `json:"partition"`
	Offset          int64         `json:"offset"`
	ConsumerGroup   string        `json:"consumer_group"`
	FailureReason   string        `json:"failure_reason"`
	FailureCategory string        `json:"failure_category"`
	AttemptCount    int           `json:"attempt_count"`
	FailedAt        time.Time     `json:"failed_at"`
}

// MarshalDLQMessage serializes a DLQ message for Kafka.
func MarshalDLQMessage(msg DLQMessage) ([]byte, error) {
	msg.FailedAt = msg.FailedAt.UTC()
	return json.Marshal(msg)
}

// UnmarshalDLQMessage deserializes a DLQ message.
func UnmarshalDLQMessage(data []byte) (DLQMessage, error) {
	var msg DLQMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return DLQMessage{}, err
	}
	return msg, nil
}

// NewDLQMessage builds a DLQ record from a consumed message.
func NewDLQMessage(envelope Envelope, topic string, partition int, offset int64, group, reason, category string, attempts int) DLQMessage {
	return DLQMessage{
		OriginalEvent:   FromEnvelope(envelope),
		OriginalTopic:   topic,
		Partition:       partition,
		Offset:          offset,
		ConsumerGroup:   group,
		FailureReason:   reason,
		FailureCategory: category,
		AttemptCount:    attempts,
		FailedAt:        time.Now().UTC(),
	}
}
