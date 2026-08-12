package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// Message is a Kafka record with platform metadata headers.
type Message struct {
	Topic          string
	Key            []byte
	Value          []byte
	EventID        string
	EventType      string
	EventVersion   string
	CorrelationID  string
	RequestID      string
	Producer       string
	TraceID        string
	SpanID         string
	Propagation    map[string]string
	Partition      int
	Offset         int64
}

// Producer publishes messages to Kafka.
type Producer interface {
	Publish(ctx context.Context, msg Message) error
	Close() error
}

// ConsumerHandler processes a Kafka message value.
type ConsumerHandler func(ctx context.Context, msg Message) error

// Consumer subscribes to Kafka topics.
type Consumer interface {
	Run(ctx context.Context, topic, groupID string, handler ConsumerHandler) error
	Close() error
}

// Config holds Kafka client settings.
type Config struct {
	Brokers  []string
	ClientID string
}

// WriterProducer implements Producer using segmentio/kafka-go.
type WriterProducer struct {
	writers map[string]*kafka.Writer
	brokers []string
}

// NewWriterProducer creates a Kafka producer.
func NewWriterProducer(cfg Config) *WriterProducer {
	return &WriterProducer{
		writers: make(map[string]*kafka.Writer),
		brokers: cfg.Brokers,
	}
}

func (p *WriterProducer) writer(topic string) *kafka.Writer {
	if w, ok := p.writers[topic]; ok {
		return w
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(p.brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	p.writers[topic] = w
	return w
}

// Publish sends a message to Kafka.
func (p *WriterProducer) Publish(ctx context.Context, msg Message) error {
	headers := []kafka.Header{
		{Key: "event_id", Value: []byte(msg.EventID)},
		{Key: "event_type", Value: []byte(msg.EventType)},
	}
	if msg.EventVersion != "" {
		headers = append(headers, kafka.Header{Key: "event_version", Value: []byte(msg.EventVersion)})
	}
	if msg.CorrelationID != "" {
		headers = append(headers, kafka.Header{Key: "correlation_id", Value: []byte(msg.CorrelationID)})
	}
	if msg.RequestID != "" {
		headers = append(headers, kafka.Header{Key: "request_id", Value: []byte(msg.RequestID)})
	}
	if msg.Producer != "" {
		headers = append(headers, kafka.Header{Key: "producer", Value: []byte(msg.Producer)})
	}
	if msg.TraceID != "" {
		headers = append(headers, kafka.Header{Key: "trace_id", Value: []byte(msg.TraceID)})
	}
	if msg.SpanID != "" {
		headers = append(headers, kafka.Header{Key: "span_id", Value: []byte(msg.SpanID)})
	}
	prop := map[string]string{}
	InjectTraceContext(ctx, prop)
	for k, v := range prop {
		headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	err := p.writer(msg.Topic).WriteMessages(ctx, kafka.Message{
		Topic:   msg.Topic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: headers,
		Time:    time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("kafka publish: %w", err)
	}
	return nil
}

// Close closes all topic writers.
func (p *WriterProducer) Close() error {
	var errs []string
	for topic, w := range p.writers {
		if err := w.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", topic, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close kafka writers: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ReaderConsumer implements Consumer using segmentio/kafka-go.
type ReaderConsumer struct {
	brokers []string
	readers []*kafka.Reader
}

// NewReaderConsumer creates a Kafka consumer.
func NewReaderConsumer(cfg Config) *ReaderConsumer {
	return &ReaderConsumer{brokers: cfg.Brokers}
}

// Run consumes messages from a topic until the context is cancelled.
func (c *ReaderConsumer) Run(ctx context.Context, topic, groupID string, handler ConsumerHandler) error {
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

	for {
		kmsg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("fetch kafka message: %w", err)
		}

		msg := DecodeMessage(kmsg)

		if err := handler(ctx, msg); err != nil {
			return fmt.Errorf("handle message: %w", err)
		}

		if err := reader.CommitMessages(ctx, kmsg); err != nil {
			return fmt.Errorf("commit kafka message: %w", err)
		}
	}
}

// Close closes all readers.
func (c *ReaderConsumer) Close() error {
	var errs []string
	for _, r := range c.readers {
		if err := r.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close kafka readers: %s", strings.Join(errs, "; "))
	}
	return nil
}

// EnsureTopic creates a topic if it does not exist (for integration tests).
func EnsureTopic(ctx context.Context, cfg Config, topic string, partitions int) error {
	if partitions <= 0 {
		partitions = 1
	}
	conn, err := kafka.DialContext(ctx, "tcp", cfg.Brokers[0])
	if err != nil {
		return fmt.Errorf("dial kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("kafka controller: %w", err)
	}

	controllerConn, err := kafka.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("dial kafka controller: %w", err)
	}
	defer controllerConn.Close()

	return controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: 1,
	})
}
