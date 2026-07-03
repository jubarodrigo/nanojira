package main

import (
	"fmt"
	"os"

	"github.com/rodrigocavalhero/nanojira/internal/config"
	"github.com/rodrigocavalhero/nanojira/internal/repository/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "migrations"
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	var runErr error
	switch command {
	case "up":
		runErr = postgres.RunMigrations(cfg.DatabaseURL, dir)
	case "down":
		runErr = postgres.MigrateDown(cfg.DatabaseURL, dir)
	case "status":
		runErr = postgres.MigrationStatus(cfg.DatabaseURL, dir)
	default:
		fmt.Fprintf(os.Stderr, "usage: migrate [up|down|status]\n")
		os.Exit(1)
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "migrate %s: %v\n", command, runErr)
		os.Exit(1)
	}
}
