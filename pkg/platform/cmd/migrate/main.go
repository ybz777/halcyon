package main

import (
	"fmt"
	"os"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
)

func main() {
	cfg, err := platformconfig.LoadMigrate("DATABASE_URL")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := migrations.Run(cfg.DatabaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("platform migrations applied")
}
