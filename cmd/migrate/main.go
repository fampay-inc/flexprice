package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/flexprice/flexprice/ent"
	entmigrate "github.com/flexprice/flexprice/ent/migrate"
	"github.com/flexprice/flexprice/internal/config"
	"github.com/flexprice/flexprice/internal/logger"
	_ "github.com/lib/pq"
)

func main() {
	// Parse command line flags
	dryRun := flag.Bool("dry-run", false, "Print migration SQL without executing it")
	timeout := flag.Int("timeout", 300, "Timeout in seconds for the migration")
	flag.Parse()

	// Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := logger.NewLogger(cfg)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// Get DSN from config
	dsn := cfg.Postgres.GetDSN()
	logger.Infow("Connecting to database", "host", cfg.Postgres.Host)

	// Create Ent client
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		logger.Fatalw("Failed to connect to postgres", "error", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	// Run auto migration
	logger.Info("Running database migrations...")

	// Tables managed by raw SQL migrations (e.g. partitioned tables) must be
	// excluded here so that Ent does not attempt to create them as plain tables,
	// which would conflict with the DDL already applied by migrate-postgres.
	rawSQLManagedTables := map[string]bool{
		"benefit_ledgers": true,
	}
	filteredTables := make([]*schema.Table, 0, len(entmigrate.Tables))
	for _, t := range entmigrate.Tables {
		if !rawSQLManagedTables[t.Name] {
			filteredTables = append(filteredTables, t)
		}
	}

	// Check if we're in dry-run mode
	if *dryRun {
		logger.Info("Dry run mode - printing migration SQL without executing")
		err = client.Schema.WriteTo(ctx, os.Stdout)
		if err != nil {
			logger.Fatalw("Failed to generate migration SQL", "error", err)
		}
	} else {
		// Run the actual migration using filtered tables only
		err = entmigrate.Create(ctx, client.Schema, filteredTables)
		if err != nil {
			logger.Fatalw("Failed to create schema resources", "error", err)
		}
		logger.Info("Migration completed successfully")
	}

	fmt.Println("Migration process completed")
}
