package cmd

import (
	"fmt"
	"log/slog"

	"github.com/shunichi-ikebuchi/accounting-system/pkg/config"
	"github.com/shunichi-ikebuchi/accounting-system/pkg/db"
	"github.com/shunichi-ikebuchi/accounting-system/pkg/pathutil"
	"github.com/spf13/cobra"
)

// statsCmd represents the stats command.
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display sync statistics",
	Long: `Display statistics about synced transactions and documents.

Shows:
- Total number of synced deals
- Total number of synced journals
- Total number of attached documents
- Last sync timestamp

Example:
  freee-sync stats`,
	Run: runStats,
}

func runStats(cmd *cobra.Command, args []string) {
	slog.Info("Loading configuration")

	// Load configuration
	cfg, err := config.Load(getConfigFile())
	exitOnError(err, "failed to load configuration")

	// Validate required fields
	if err := cfg.Validate([]string{"beancount", "root"}); err != nil {
		exitOnError(err, "invalid configuration")
	}

	// Initialize PathResolver
	pathResolver := pathutil.New(pathutil.Config{
		BeancountRoot:  cfg.Beancount.Root,
		DatabasePath:   cfg.Beancount.DBPath,
		AttachmentsDir: cfg.Beancount.AttachmentsDir,
	})

	// Open database connection
	dbPath := pathResolver.GetDatabasePath()
	slog.Debug("Opening database", "path", dbPath)

	conn, err := db.Open(dbPath)
	exitOnError(err, "failed to open database")
	defer conn.Close()

	// Get sync history
	syncHistory := db.NewSyncHistory(conn)

	// Get statistics
	stats, err := syncHistory.GetStats()
	exitOnError(err, "failed to get statistics")

	// Display statistics
	fmt.Println("\n=== Sync Statistics ===")
	fmt.Printf("Total synced deals:    %d\n", stats.TotalDeals)
	fmt.Printf("Total synced journals: %d\n", stats.TotalJournals)
	fmt.Printf("Total documents:       %d\n", stats.TotalDocuments)

	if stats.LastSync.Valid {
		fmt.Printf("Last sync:             %s\n", stats.LastSync.String)
	} else {
		fmt.Printf("Last sync:             (never)\n")
	}

	fmt.Println()

	slog.Info("Statistics displayed successfully")
}
