package cmd

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/shunichi-ikebuchi/accounting-system/pkg/beancount"
	"github.com/shunichi-ikebuchi/accounting-system/pkg/config"
	"github.com/shunichi-ikebuchi/accounting-system/pkg/converter"
	"github.com/shunichi-ikebuchi/accounting-system/pkg/db"
	"github.com/shunichi-ikebuchi/accounting-system/pkg/freee"
	"github.com/shunichi-ikebuchi/accounting-system/pkg/pathutil"
	"github.com/spf13/cobra"
)

var (
	dateFrom string
	dateTo   string
	dryRun   bool
)

// syncCmd represents the sync command.
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync freee transactions to Beancount",
	Long: `Sync transactions from freee Accounting API to Beancount files.

This command:
1. Fetches deals and journals from freee API
2. Filters out already synced items
3. Converts them to Beancount format
4. Appends to monthly Beancount files
5. Records sync history in SQLite

Example:
  freee-sync sync --from 2024-01-01 --to 2024-01-31
  freee-sync sync --from 2024-01-01 --to 2024-01-31 --dry-run`,
	Run: runSync,
}

func init() {
	// Flags
	syncCmd.Flags().StringVar(&dateFrom, "from", "", "Start date (YYYY-MM-DD) (required)")
	syncCmd.Flags().StringVar(&dateTo, "to", "", "End date (YYYY-MM-DD) (required)")
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run mode (no file writes)")

	syncCmd.MarkFlagRequired("from")
	syncCmd.MarkFlagRequired("to")
}

func runSync(cmd *cobra.Command, args []string) {
	slog.Info("Starting sync", "from", dateFrom, "to", dateTo, "dry_run", dryRun)

	// Load configuration
	cfg, err := config.Load(getConfigFile())
	exitOnError(err, "failed to load configuration")

	// Validate required fields
	if err := cfg.Validate(
		[]string{"freee", "apiUrl"},
		[]string{"freee", "accessToken"},
		[]string{"freee", "companyId"},
		[]string{"beancount", "root"},
	); err != nil {
		exitOnError(err, "invalid configuration")
	}

	// Initialize components
	pathResolver := pathutil.New(pathutil.Config{
		BeancountRoot:  cfg.Beancount.Root,
		DatabasePath:   cfg.Beancount.DBPath,
		AttachmentsDir: cfg.Beancount.AttachmentsDir,
	})

	// Open database
	dbPath := pathResolver.GetDatabasePath()
	slog.Debug("Opening database", "path", dbPath)
	conn, err := db.Open(dbPath)
	exitOnError(err, "failed to open database")
	defer conn.Close()

	syncHistory := db.NewSyncHistory(conn)

	// Initialize freee API client
	freeeClient := freee.NewClient(freee.ClientConfig{
		APIURL:      cfg.Freee.APIURL,
		AccessToken: cfg.Freee.AccessToken,
		CompanyID:   cfg.Freee.CompanyID,
		Timeout:     30 * time.Second,
	})

	// Initialize account mapper
	mappingFilePath := filepath.Join("config", "account-mapping.yaml")
	mapper, err := converter.NewMapper(mappingFilePath)
	exitOnError(err, "failed to load account mapping")

	// Initialize converter
	cvtr := converter.NewConverter(mapper, "JPY")

	// Initialize Beancount repository
	beancountRepo := beancount.NewFileSystemRepository(pathResolver)

	// Fetch deals from freee
	slog.Info("Fetching deals from freee", "from", dateFrom, "to", dateTo)
	allDeals, err := freeeClient.FetchAllDeals(dateFrom, dateTo)
	exitOnError(err, "failed to fetch deals")
	slog.Info("Fetched deals", "count", len(allDeals))

	// Fetch journals from freee
	slog.Info("Fetching journals from freee", "from", dateFrom, "to", dateTo)
	allJournals, err := freeeClient.FetchAllJournals(dateFrom, dateTo)
	exitOnError(err, "failed to fetch journals")
	slog.Info("Fetched journals", "count", len(allJournals))

	// Filter out already synced items
	slog.Info("Checking for already synced items")
	syncedDealIDs, err := syncHistory.GetSyncedIDs(db.SyncTypeDeal)
	exitOnError(err, "failed to get synced deal IDs")

	syncedJournalIDs, err := syncHistory.GetSyncedIDs(db.SyncTypeJournal)
	exitOnError(err, "failed to get synced journal IDs")

	newDeals := filterDeals(allDeals, syncedDealIDs)
	newJournals := filterJournals(allJournals, syncedJournalIDs)

	slog.Info("New items to sync",
		"new_deals", len(newDeals),
		"new_journals", len(newJournals),
		"skipped_deals", len(allDeals)-len(newDeals),
		"skipped_journals", len(allJournals)-len(newJournals),
	)

	if len(newDeals) == 0 && len(newJournals) == 0 {
		fmt.Println("No new items to sync")
		return
	}

	// Group by month
	dealsByMonth := groupDealsByMonth(newDeals)
	journalsByMonth := groupJournalsByMonth(newJournals)

	// Get all unique months
	allMonths := getAllMonths(dealsByMonth, journalsByMonth)

	filesWritten := []string{}

	// Process each month
	for _, monthKey := range allMonths {
		monthDeals := dealsByMonth[monthKey]
		monthJournals := journalsByMonth[monthKey]

		filePath, err := pathResolver.GetMonthFilePath(monthKey)
		if err != nil {
			slog.Error("Failed to get month file path", "month", monthKey, "error", err)
			continue
		}

		if !dryRun {
			// Ensure month file exists
			if err := beancountRepo.EnsureMonthFile(monthKey); err != nil {
				slog.Error("Failed to ensure month file", "month", monthKey, "error", err)
				continue
			}

			// Append transactions
			for _, deal := range monthDeals {
				txn := cvtr.ConvertDeal(deal)
				formatted := cvtr.FormatTransaction(txn)

				if err := beancountRepo.AppendTransaction(monthKey, formatted); err != nil {
					slog.Error("Failed to append deal", "deal_id", deal.ID, "error", err)
					continue
				}

				// Record sync history
				if err := syncHistory.RecordSync(db.SyncRecord{
					SyncType:      db.SyncTypeDeal,
					FreeeID:       deal.ID,
					IssueDate:     deal.IssueDate,
					Amount:        deal.Amount,
					BeancountFile: filePath,
				}); err != nil {
					slog.Error("Failed to record sync", "deal_id", deal.ID, "error", err)
				}
			}

			for _, journal := range monthJournals {
				txn := cvtr.ConvertJournal(journal)
				formatted := cvtr.FormatTransaction(txn)

				if err := beancountRepo.AppendTransaction(monthKey, formatted); err != nil {
					slog.Error("Failed to append journal", "journal_id", journal.ID, "error", err)
					continue
				}

				amount := int64(0)
				if len(journal.Details) > 0 {
					amount = journal.Details[0].Amount
				}

				// Record sync history
				if err := syncHistory.RecordSync(db.SyncRecord{
					SyncType:      db.SyncTypeJournal,
					FreeeID:       journal.ID,
					IssueDate:     journal.IssueDate,
					Amount:        amount,
					BeancountFile: filePath,
				}); err != nil {
					slog.Error("Failed to record sync", "journal_id", journal.ID, "error", err)
				}
			}

			filesWritten = append(filesWritten, filePath)
			slog.Info("Updated file",
				"path", filePath,
				"deals", len(monthDeals),
				"journals", len(monthJournals),
			)
		} else {
			// Dry run: print transactions
			fmt.Printf("[DRY RUN] Would append to %s\n", filePath)
			for _, deal := range monthDeals {
				txn := cvtr.ConvertDeal(deal)
				fmt.Println(cvtr.FormatTransaction(txn))
			}
			for _, journal := range monthJournals {
				txn := cvtr.ConvertJournal(journal)
				fmt.Println(cvtr.FormatTransaction(txn))
			}
		}
	}

	// Display final statistics
	if !dryRun {
		stats, err := syncHistory.GetStats()
		if err == nil {
			fmt.Println("\n=== Sync Statistics ===")
			fmt.Printf("Total synced deals:    %d\n", stats.TotalDeals)
			fmt.Printf("Total synced journals: %d\n", stats.TotalJournals)
			fmt.Printf("Total documents:       %d\n", stats.TotalDocuments)
			if stats.LastSync.Valid {
				fmt.Printf("Last sync:             %s\n", stats.LastSync.String)
			}
			fmt.Println()
		}
	}

	slog.Info("Sync completed",
		"new_deals", len(newDeals),
		"new_journals", len(newJournals),
		"files_written", len(filesWritten),
	)
}

// Helper functions

func filterDeals(deals []freee.Deal, syncedIDs []int64) []freee.Deal {
	syncedIDMap := make(map[int64]bool)
	for _, id := range syncedIDs {
		syncedIDMap[id] = true
	}

	var result []freee.Deal
	for _, deal := range deals {
		if !syncedIDMap[deal.ID] {
			result = append(result, deal)
		}
	}
	return result
}

func filterJournals(journals []freee.Journal, syncedIDs []int64) []freee.Journal {
	syncedIDMap := make(map[int64]bool)
	for _, id := range syncedIDs {
		syncedIDMap[id] = true
	}

	var result []freee.Journal
	for _, journal := range journals {
		if !syncedIDMap[journal.ID] {
			result = append(result, journal)
		}
	}
	return result
}

func groupDealsByMonth(deals []freee.Deal) map[string][]freee.Deal {
	groups := make(map[string][]freee.Deal)
	for _, deal := range deals {
		monthKey := deal.IssueDate[:7] // YYYY-MM
		groups[monthKey] = append(groups[monthKey], deal)
	}
	return groups
}

func groupJournalsByMonth(journals []freee.Journal) map[string][]freee.Journal {
	groups := make(map[string][]freee.Journal)
	for _, journal := range journals {
		monthKey := journal.IssueDate[:7] // YYYY-MM
		groups[monthKey] = append(groups[monthKey], journal)
	}
	return groups
}

func getAllMonths(dealGroups map[string][]freee.Deal, journalGroups map[string][]freee.Journal) []string {
	monthsMap := make(map[string]bool)
	for month := range dealGroups {
		monthsMap[month] = true
	}
	for month := range journalGroups {
		monthsMap[month] = true
	}

	months := []string{}
	for month := range monthsMap {
		months = append(months, month)
	}

	// Sort months (simple string sort works for YYYY-MM format)
	for i := 0; i < len(months); i++ {
		for j := i + 1; j < len(months); j++ {
			if months[i] > months[j] {
				months[i], months[j] = months[j], months[i]
			}
		}
	}

	return months
}
