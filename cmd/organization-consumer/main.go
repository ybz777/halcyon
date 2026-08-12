package main

import (
	"log/slog"
	"os"
)

func main() {
	slog.Warn("organization-audit-consumer is deprecated; use services/audit/cmd/consumer (audit-log-consumer)")
	os.Exit(0)
}
