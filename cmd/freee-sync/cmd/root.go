// Package cmd provides CLI commands for freee-sync.
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	debug   bool
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "freee-sync",
	Short: "Sync freee accounting data to Beancount",
	Long: `freee-sync is a CLI tool that synchronizes transaction data
from freee Accounting API to Beancount plain-text accounting files.

It supports:
- Syncing deals and journals from freee
- Converting to Beancount format
- Preventing duplicate syncs with SQLite history
- Dry-run mode for testing

Example:
  freee-sync sync --from 2024-01-01 --to 2024-01-31
  freee-sync stats`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Setup logging
		logLevel := slog.LevelInfo
		if debug {
			logLevel = slog.LevelDebug
		}

		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: logLevel,
		}))
		slog.SetDefault(logger)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .env)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")

	// Add subcommands
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(statsCmd)
}

// Helper function to get config file path.
func getConfigFile() string {
	if cfgFile != "" {
		return cfgFile
	}
	return "" // Will use default .env loading
}

// Helper function to handle errors and exit.
func exitOnError(err error, msg string) {
	if err != nil {
		slog.Error(msg, "error", err)
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", msg, err)
		os.Exit(1)
	}
}
