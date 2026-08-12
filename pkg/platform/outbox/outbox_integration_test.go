//go:build integration

package outbox_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/events"
	platformkafka "github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	"github.com/halcyon-io/halcyon/pkg/platform/outbox"
	"github.com/jackc/pgx/v5/pgxpool"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupMessagingTest(t *testing.T) (*pgxpool.Pool, platformkafka.Config, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("halcyon"),
		tcpostgres.WithUsername("halcyon"),
		tcpostgres.WithPassword("halcyon"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	if err := migrations.Run(connStr); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	kafkaContainer, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.5.0")
	if err != nil {
		t.Fatalf("start kafka: %v", err)
	}

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		t.Fatalf("kafka brokers: %v", err)
	}

	cfg := platformkafka.Config{Brokers: brokers}
	cleanup := func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
		_ = kafkaContainer.Terminate(ctx)
	}

	return pool, cfg, cleanup
}

func TestOutboxRelayPublishesToKafka(t *testing.T) {
	pool, kafkaCfg, cleanup := setupMessagingTest(t)
	defer cleanup()

	ctx := context.Background()
	topic := "halcyon.auth"
	if err := platformkafka.EnsureTopic(ctx, kafkaCfg, topic, 1); err != nil {
		t.Fatalf("ensure topic: %v", err)
	}
	if err := platformkafka.EnsureTopic(ctx, kafkaCfg, events.DLQTopicFor(topic), 1); err != nil {
		t.Fatalf("ensure dlq topic: %v", err)
	}

	store := outbox.NewPostgresStore(pool)
	producer := platformkafka.NewWriterProducer(kafkaCfg)
	defer producer.Close()

	record, err := events.BuildRecord(
		events.TypeUserRegistered,
		"usr_integrationtest01",
		"user",
		events.SystemOrganizationID,
		time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
		events.UserRegisteredPayload{UserID: "usr_integrationtest01", Email: "relay@example.com"},
	)
	if err != nil {
		t.Fatalf("build record: %v", err)
	}

	pub := outbox.NewPublisher(store, "integration-test")
	if err := pub.Publish(ctx, record); err != nil {
		t.Fatalf("publish to outbox: %v", err)
	}

	relay := outbox.NewRelay(store, producer, messaging.NewMetrics(), outbox.RelayConfig{
		BatchSize:    10,
		PollInterval: time.Millisecond,
		MaxAttempts:  3,
	}, nil)
	if err := relay.ProcessBatch(ctx); err != nil {
		t.Fatalf("relay batch: %v", err)
	}

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: kafkaCfg.Brokers,
		Topic:   topic,
		MaxWait: time.Second,
	})
	defer reader.Close()

	msgCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	msg, err := reader.ReadMessage(msgCtx)
	if err != nil {
		t.Fatalf("read kafka message: %v", err)
	}

	envelope, err := events.DeserializeEnvelope(msg.Value)
	if err != nil {
		t.Fatalf("deserialize envelope: %v", err)
	}
	if envelope.EventType != events.TypeUserRegistered {
		t.Fatalf("event type = %q", envelope.EventType)
	}
}

func TestConsumerIdempotencyAndDLQ(t *testing.T) {
	pool, kafkaCfg, cleanup := setupMessagingTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	topic := "halcyon.organizations"
	if err := platformkafka.EnsureTopic(ctx, kafkaCfg, topic, 1); err != nil {
		t.Fatalf("ensure topic: %v", err)
	}
	dlqTopic := events.DLQTopicFor(topic)
	if err := platformkafka.EnsureTopic(ctx, kafkaCfg, dlqTopic, 1); err != nil {
		t.Fatalf("ensure dlq topic: %v", err)
	}

	producer := platformkafka.NewWriterProducer(kafkaCfg)
	defer producer.Close()

	envelope := events.Envelope{
		EventID:        "evt_idempotencytest01",
		EventType:      events.TypeOrganizationCreated,
		EventVersion:   events.VersionV1,
		AggregateID:    "org_integrationtest01",
		AggregateType:  "organization",
		OrganizationID: "org_integrationtest01",
		CorrelationID:  "evt_idempotencytest01",
		OccurredAt:     time.Now().UTC(),
		Producer:       "integration-test",
		Payload:        mustMarshalPayload(t, events.TypeOrganizationCreated, events.OrganizationCreatedPayload{OrganizationID: "org_integrationtest01", Name: "Acme", Slug: "acme"}),
	}
	data, err := events.SerializeEnvelope(envelope)
	if err != nil {
		t.Fatalf("serialize envelope: %v", err)
	}

	if err := producer.Publish(ctx, platformkafka.Message{
		Topic:     topic,
		Key:       []byte(envelope.OrganizationID),
		Value:     data,
		EventID:   envelope.EventID,
		EventType: envelope.EventType,
	}); err != nil {
		t.Fatalf("publish message: %v", err)
	}

	processed := messaging.NewPostgresProcessedStore(pool)
	metrics := messaging.NewMetrics()
	attempts := 0

	handler := events.HandlerFunc(func(ctx context.Context, env events.Envelope) error {
		attempts++
		if attempts < 3 {
			return errRetry
		}
		return nil
	})

	consumer := platformkafka.NewReaderConsumer(kafkaCfg)
	defer consumer.Close()

	pipeline := messaging.NewPipeline(consumer, messaging.PipelineConfig{
		Topic:     topic,
		GroupID:   "integration-test-group",
		Retry:     messaging.RetryConfig{MaxAttempts: 3, Initial: 10 * time.Millisecond, MaxBackoff: 50 * time.Millisecond, Multiplier: 2},
		Processed: processed,
		Metrics:   metrics,
		Producer:  producer,
	})

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- pipeline.Run(runCtx, handler)
	}()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		ok, err := processed.IsProcessed(ctx, envelope.EventID, "integration-test-group")
		if err != nil {
			t.Fatalf("is processed: %v", err)
		}
		if ok {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	ok, err := processed.IsProcessed(ctx, envelope.EventID, "integration-test-group")
	if err != nil {
		t.Fatalf("is processed: %v", err)
	}
	if !ok {
		t.Fatal("event was not marked processed")
	}

	attemptsBefore := attempts
	if err := producer.Publish(ctx, platformkafka.Message{
		Topic: topic, Key: []byte(envelope.OrganizationID), Value: data,
		EventID: envelope.EventID, EventType: envelope.EventType,
	}); err != nil {
		t.Fatalf("republish message: %v", err)
	}

	time.Sleep(2 * time.Second)
	if attempts != attemptsBefore {
		t.Fatalf("duplicate increased attempts: before=%d after=%d", attemptsBefore, attempts)
	}

	runCancel()
	<-errCh
}

func TestIdempotentProcessorConcurrentAndCrossGroup(t *testing.T) {
	pool, kafkaCfg, cleanup := setupMessagingTest(t)
	defer cleanup()

	ctx := context.Background()
	processor := messaging.NewIdempotentProcessor(pool)
	var calls atomic.Int32

	fn := func(ctx context.Context) error {
		calls.Add(1)
		return nil
	}

	eventID := "evt_idemconctest00001"
	orgID := "org_integrationtest02"

	done := make(chan error, 2)
	go func() { done <- runProcessOnce(processor, ctx, eventID, "group-a", orgID, fn) }()
	go func() { done <- runProcessOnce(processor, ctx, eventID, "group-a", orgID, fn) }()
	<-done
	<-done

	if calls.Load() != 1 {
		t.Fatalf("concurrent group-a calls = %d, want 1", calls.Load())
	}

	result, err := processor.ProcessOnce(ctx, eventID, "group-b", orgID, fn)
	if err != nil {
		t.Fatalf("group-b process: %v", err)
	}
	if result.Duplicate {
		t.Fatal("group-b should process independently")
	}
	if calls.Load() != 2 {
		t.Fatalf("total calls = %d, want 2", calls.Load())
	}
}

func runProcessOnce(p *messaging.IdempotentProcessor, ctx context.Context, eventID, group, orgID string, fn func(context.Context) error) error {
	_, err := p.ProcessOnce(ctx, eventID, group, orgID, fn)
	return err
}

func TestEndToEndOutboxConsumerIdempotency(t *testing.T) {
	pool, kafkaCfg, cleanup := setupMessagingTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	topic := "halcyon.auth"
	if err := platformkafka.EnsureTopic(ctx, kafkaCfg, topic, 1); err != nil {
		t.Fatalf("ensure topic: %v", err)
	}

	store := outbox.NewPostgresStore(pool)
	producer := platformkafka.NewWriterProducer(kafkaCfg)
	defer producer.Close()

	record, err := events.BuildRecord(
		events.TypeUserRegistered,
		"usr_e2etest00000001",
		"user",
		events.SystemOrganizationID,
		time.Now().UTC(),
		events.UserRegisteredPayload{UserID: "usr_e2etest00000001", Email: "e2e@example.com"},
	)
	if err != nil {
		t.Fatalf("build record: %v", err)
	}

	pub := outbox.NewPublisher(store, "integration-test")
	if err := pub.Publish(ctx, record); err != nil {
		t.Fatalf("publish outbox: %v", err)
	}

	relay := outbox.NewRelay(store, producer, messaging.NewMetrics(), outbox.RelayConfig{
		BatchSize: 10, MaxAttempts: 3,
	}, nil)
	if err := relay.ProcessBatch(ctx); err != nil {
		t.Fatalf("relay: %v", err)
	}

	var eventID string
	var payloadBytes []byte
	err = pool.QueryRow(ctx, `
		SELECT event_id, payload FROM outbox WHERE aggregate_id = $1 AND status = 'published' LIMIT 1
	`, record.AggregateID()).Scan(&eventID, &payloadBytes)
	if err != nil {
		t.Fatalf("query outbox: %v", err)
	}

	consumer := platformkafka.NewReaderConsumer(kafkaCfg)
	defer consumer.Close()

	var handled atomic.Int32
	handler := events.HandlerFunc(func(ctx context.Context, env events.Envelope) error {
		handled.Add(1)
		return nil
	})

	pipeline := messaging.NewPipeline(consumer, messaging.PipelineConfig{
		Topic:      topic,
		GroupID:    "e2e-auth-consumer",
		Idempotent: messaging.NewIdempotentProcessor(pool),
		Producer:   producer,
	})

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()
	errCh := make(chan error, 1)
	go func() { errCh <- pipeline.Run(runCtx, handler) }()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) && handled.Load() == 0 {
		time.Sleep(100 * time.Millisecond)
	}
	if handled.Load() != 1 {
		t.Fatalf("handled = %d, want 1", handled.Load())
	}

	// Republish the same event to Kafka to verify idempotent consumption.
	data, err := events.SerializeEnvelope(events.Envelope{
		EventID: eventID, EventType: record.EventType(), EventVersion: record.EventVersion(),
		AggregateID: record.AggregateID(), AggregateType: record.AggregateType(),
		OrganizationID: record.OrganizationID(), OccurredAt: record.OccurredAt(),
		Producer: "integration-test", Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	before := handled.Load()
	if err := producer.Publish(ctx, platformkafka.Message{
		Topic: topic, Key: []byte(record.AggregateID()), Value: data,
		EventID: eventID, EventType: record.EventType(),
	}); err != nil {
		t.Fatalf("republish: %v", err)
	}
	time.Sleep(2 * time.Second)
	if handled.Load() != before {
		t.Fatalf("duplicate handling: before=%d after=%d", before, handled.Load())
	}

	runCancel()
	<-errCh
}

var errRetry = &retryError{}

type retryError struct{}

func (e *retryError) Error() string { return "retry" }

func mustMarshalPayload(t *testing.T, eventType string, payload any) []byte {
	t.Helper()
	data, err := events.MarshalPayload(eventType, payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return data
}
