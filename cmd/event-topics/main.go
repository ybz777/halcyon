package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
)

func main() {
	env := platformconfig.FromEnv()
	brokers := splitCSV(env.String("KAFKA_BROKERS", "localhost:9092"))
	if err := kafka.EnsurePlatformTopics(context.Background(), kafka.Config{
		Brokers:  brokers,
		ClientID: env.String("KAFKA_CLIENT_ID", "event-topics"),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "ensure topics: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("platform topics ensured")
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
