package kafka

import (
	"context"
	"fmt"

	"github.com/halcyon-io/halcyon/pkg/platform/eventtopics"
)

// TopicSpec describes a platform Kafka topic.
type TopicSpec struct {
	Name       string
	Partitions int
}

// PlatformTopics returns required Halcyon platform topics.
func PlatformTopics() []TopicSpec {
	names := []string{
		eventtopics.TopicForEventType("auth.user.registered"),
		eventtopics.TopicForEventType("organizations.organization.created"),
		eventtopics.TopicForEventType("workflow.execution.started"),
		eventtopics.TopicForEventType("security.invalid_token"),
	}
	seen := make(map[string]struct{}, len(names)*2)
	specs := make([]TopicSpec, 0, len(names)*2)
	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		specs = append(specs, TopicSpec{Name: name, Partitions: 6})
		dlq := eventtopics.DLQTopicFor(name)
		if _, ok := seen[dlq]; !ok {
			seen[dlq] = struct{}{}
			specs = append(specs, TopicSpec{Name: dlq, Partitions: 3})
		}
	}
	return specs
}

// EnsurePlatformTopics creates all platform topics for local development.
func EnsurePlatformTopics(ctx context.Context, cfg Config) error {
	for _, spec := range PlatformTopics() {
		if err := EnsureTopic(ctx, cfg, spec.Name, spec.Partitions); err != nil {
			return fmt.Errorf("ensure topic %s: %w", spec.Name, err)
		}
	}
	return nil
}
