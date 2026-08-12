package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// ConcurrentConsumer supports bounded parallel message handling.
type ConcurrentConsumer interface {
	Consumer
	RunConcurrent(ctx context.Context, topic, groupID string, concurrency int, handler ConsumerHandler) error
}

// RunConcurrent consumes messages with a bounded worker pool.
func (c *ReaderConsumer) RunConcurrent(ctx context.Context, topic, groupID string, concurrency int, handler ConsumerHandler) error {
	if concurrency <= 1 {
		return c.Run(ctx, topic, groupID, handler)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})
	c.readers = append(c.readers, reader)

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	for {
		kmsg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				wg.Wait()
				return ctx.Err()
			}
			return fmt.Errorf("fetch kafka message: %w", err)
		}

		msg := DecodeMessage(kmsg)
		sem <- struct{}{}
		wg.Add(1)
		go func(kmsg kafka.Message, msg Message) {
			defer func() {
				<-sem
				wg.Done()
			}()
			if err := handler(ctx, msg); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			if err := reader.CommitMessages(ctx, kmsg); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(kmsg, msg)

		select {
		case err := <-errCh:
			return err
		default:
		}
	}
}

// DecodeMessage maps a kafka-go record to platform metadata.
func DecodeMessage(kmsg kafka.Message) Message {
	msg := Message{
		Topic:     kmsg.Topic,
		Key:       kmsg.Key,
		Value:     kmsg.Value,
		Partition: kmsg.Partition,
		Offset:    kmsg.Offset,
	}
	for _, h := range kmsg.Headers {
		switch h.Key {
		case "event_id":
			msg.EventID = string(h.Value)
		case "event_type":
			msg.EventType = string(h.Value)
		case "event_version":
			msg.EventVersion = string(h.Value)
		case "correlation_id":
			msg.CorrelationID = string(h.Value)
		case "request_id":
			msg.RequestID = string(h.Value)
		case "producer":
			msg.Producer = string(h.Value)
		case "trace_id":
			msg.TraceID = string(h.Value)
		case "span_id":
			msg.SpanID = string(h.Value)
		default:
			if msg.Propagation == nil {
				msg.Propagation = make(map[string]string)
			}
			msg.Propagation[h.Key] = string(h.Value)
		}
	}
	return msg
}
