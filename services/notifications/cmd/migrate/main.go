package main

import (
	"fmt"
	"os"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/postgres"
)

func main() {
	cfg, err := platformconfig.LoadMigrate("NOTIFICATIONS_DATABASE_URL")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("migrations applied")
}
