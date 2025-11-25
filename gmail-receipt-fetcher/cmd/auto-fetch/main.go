// Package main provides CLI for auto-fetching receipts based on freee unregistered transactions.
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/shunichi-ikebuchi/accounting-system/gmail-receipt-fetcher/internal/freee"
	"github.com/shunichi-ikebuchi/accounting-system/gmail-receipt-fetcher/internal/gmail"
	"github.com/shunichi-ikebuchi/accounting-system/gmail-receipt-fetcher/internal/receipt"
)

// receiptWithSearcher pairs a receipt with its corresponding Gmail searcher
type receiptWithSearcher struct {
	Receipt  *receipt.Receipt
	Searcher *receipt.Searcher
}

// transactionPair holds both the simplified Transaction and original WalletTransaction
type transactionPair struct {
	Transaction   freee.Transaction
	WalletTxn     freee.WalletTransaction
}

func main() {
	// Parse flags
	freeeAPI := flag.String("freee-api", "", "freee API URL (required, or FREEE_API_URL env)")
	freeeToken := flag.String("freee-token", "", "freee access token (or FREEE_ACCESS_TOKEN env)")
	freeeCompany := flag.String("freee-company", "", "freee company ID (or FREEE_COMPANY_ID env)")
	freeeClientID := flag.String("freee-client-id", "", "freee OAuth client ID (or FREEE_CLIENT_ID env)")
	freeeClientSecret := flag.String("freee-client-secret", "", "freee OAuth client secret (or FREEE_CLIENT_SECRET env)")
	freeeTokenPath := flag.String("freee-token-path", "", "freee token file path (or FREEE_TOKEN_PATH env)")
	gmailAPI := flag.String("gmail-api", "", "Gmail API URL (or GMAIL_API_URL env)")
	gmailLabel := flag.String("label", "", "Gmail label to search (or GMAIL_LABEL env)")
	vendors := flag.String("vendors", "", "Comma-separated vendor names (or RECEIPT_VENDORS env)")
	outputDir := flag.String("output", "", "Output directory (or RECEIPT_OUTPUT_DIR env, default: ./receipts)")
	credPath := flag.String("credentials", "", "Comma-separated Gmail OAuth credentials (or GMAIL_CREDENTIALS_PATH env)")
	tokenDir := flag.String("token-dir", "", "Directory for Gmail OAuth tokens (or GMAIL_TOKEN_DIR env)")
	toleranceDays := flag.Int("tolerance", 0, "Date tolerance in days (or RECEIPT_TOLERANCE_DAYS env, default: 3)")
	dryRun := flag.Bool("dry-run", false, "Preview without downloading")
	createDeals := flag.Bool("create-deals", false, "Create deals in freee for matched transactions")
	help := flag.Bool("help", false, "Show help")
	flag.BoolVar(help, "h", false, "Show help")

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Apply environment variable defaults (flag takes precedence)
	*freeeAPI = envOrDefault(*freeeAPI, "FREEE_API_URL", "https://api.freee.co.jp")
	*freeeToken = envOrDefault(*freeeToken, "FREEE_ACCESS_TOKEN", "")
	*freeeCompany = envOrDefault(*freeeCompany, "FREEE_COMPANY_ID", "")
	*freeeClientID = envOrDefault(*freeeClientID, "FREEE_CLIENT_ID", "")
	*freeeClientSecret = envOrDefault(*freeeClientSecret, "FREEE_CLIENT_SECRET", "")
	*freeeTokenPath = envOrDefault(*freeeTokenPath, "FREEE_TOKEN_PATH", "")
	*gmailAPI = envOrDefault(*gmailAPI, "GMAIL_API_URL", "")
	*gmailLabel = envOrDefault(*gmailLabel, "GMAIL_LABEL", "")
	*vendors = envOrDefault(*vendors, "RECEIPT_VENDORS", "")
	*outputDir = envOrDefault(*outputDir, "RECEIPT_OUTPUT_DIR", "./receipts")
	*credPath = envOrDefault(*credPath, "GMAIL_CREDENTIALS_PATH", "credentials.json")
	if *toleranceDays == 0 {
		*toleranceDays = envOrDefaultInt("RECEIPT_TOLERANCE_DAYS", 3)
	}

	// Validate required values
	// Either freee-token or (freee-client-id + freee-client-secret) must be provided
	useTokenManager := *freeeClientID != "" && *freeeClientSecret != ""
	if *freeeToken == "" && !useTokenManager {
		fmt.Fprintln(os.Stderr, "Error: --freee-token or (--freee-client-id + --freee-client-secret) is required")
		os.Exit(1)
	}

	// Set token directory from env or default
	*tokenDir = envOrDefault(*tokenDir, "GMAIL_TOKEN_DIR", "")
	if *tokenDir == "" {
		home, _ := os.UserHomeDir()
		*tokenDir = filepath.Join(home, ".config", "gmail-fetcher")
	}

	// Parse credentials paths (comma-separated)
	credentialsList := parseCredentialsPaths(*credPath)

	fmt.Println("=== Gmail Receipt Auto-Fetcher ===")
	fmt.Println()
	fmt.Printf("freee API: %s\n", *freeeAPI)
	if *gmailLabel != "" {
		fmt.Printf("Gmail label: %s\n", *gmailLabel)
	}
	if *vendors != "" {
		fmt.Printf("Vendor filter: %s\n", *vendors)
	}
	fmt.Printf("Date tolerance: %d days\n", *toleranceDays)
	if *dryRun {
		fmt.Println("Mode: DRY RUN")
	}
	fmt.Println()

	// Step 1: Fetch unregistered transactions from freee
	fmt.Println("Fetching unregistered transactions from freee...")
	var freeeClient *freee.Client
	var err error
	if useTokenManager {
		tokenManager := freee.NewTokenManager(*freeeClientID, *freeeClientSecret, *freeeTokenPath)
		freeeClient, err = freee.NewClientWithTokenManager(*freeeAPI, tokenManager)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create freee client: %v\n", err)
			os.Exit(1)
		}
	} else {
		freeeClient = freee.NewClient(*freeeAPI, *freeeToken, *freeeCompany)
	}
	walletTxns, err := freeeClient.FetchUnregisteredTransactions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to fetch from freee: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d unregistered transactions\n", len(walletTxns))

	// Filter transactions and keep original WalletTransaction for deal creation
	vendorList := parseVendors(*vendors)
	var transactionPairs []transactionPair
	var filteredTxns []freee.Transaction
	if len(vendorList) > 0 {
		filteredTxns = freee.FilterByVendor(walletTxns, vendorList)
	} else {
		filteredTxns = freee.FilterAll(walletTxns)
	}

	// Build pairs mapping Transaction ID to WalletTransaction
	walletTxnMap := make(map[int]freee.WalletTransaction)
	for _, wt := range walletTxns {
		walletTxnMap[wt.ID] = wt
	}
	for _, txn := range filteredTxns {
		if wt, ok := walletTxnMap[txn.ID]; ok {
			transactionPairs = append(transactionPairs, transactionPair{
				Transaction: txn,
				WalletTxn:   wt,
			})
		}
	}

	fmt.Printf("Filtered to %d expense transactions\n", len(transactionPairs))

	if len(transactionPairs) == 0 {
		fmt.Println("No transactions to process")
		return
	}

	// Display transactions
	fmt.Println()
	fmt.Println("Unregistered transactions:")
	for _, pair := range transactionPairs {
		fmt.Printf("  %s  ¥%s  %s\n", pair.Transaction.Date.Format("2006-01-02"), formatAmount(pair.Transaction.Amount), pair.Transaction.Description)
	}
	fmt.Println()

	// Calculate date range from transactions
	minDate := transactionPairs[0].Transaction.Date
	maxDate := transactionPairs[0].Transaction.Date
	for _, pair := range transactionPairs {
		if pair.Transaction.Date.Before(minDate) {
			minDate = pair.Transaction.Date
		}
		if pair.Transaction.Date.After(maxDate) {
			maxDate = pair.Transaction.Date
		}
	}
	fromDate := minDate.AddDate(0, 0, -*toleranceDays)
	toDate := maxDate.AddDate(0, 0, *toleranceDays)

	// Check credentials files (only required when not using emulator)
	if *gmailAPI == "" {
		for _, cp := range credentialsList {
			if _, err := os.Stat(cp); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error: credentials file not found: %s\n", cp)
				fmt.Fprintln(os.Stderr, "Run gmail-fetcher --help for setup instructions")
				os.Exit(1)
			}
		}
	}

	// Step 2: Search Gmail for receipts from all accounts
	fmt.Printf("Searching Gmail (date range: %s to %s)...\n", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	fmt.Printf("Checking %d Gmail account(s)\n", len(credentialsList))

	ctx := context.Background()
	var allReceipts []receiptWithSearcher

	for i, cp := range credentialsList {
		// Generate token path based on credentials filename
		tokenPath := filepath.Join(*tokenDir, tokenFilename(cp))

		if *gmailAPI != "" {
			fmt.Printf("[%d/%d] Initializing Gmail client (emulator: %s)...\n", i+1, len(credentialsList), *gmailAPI)
		} else {
			fmt.Printf("[%d/%d] Initializing Gmail client (%s)...\n", i+1, len(credentialsList), filepath.Base(cp))
		}

		gmailClient, err := gmail.NewClient(ctx, gmail.Config{
			CredentialsPath: cp,
			TokenPath:       tokenPath,
			APIEndpoint:     *gmailAPI,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create Gmail client for %s: %v\n", cp, err)
			continue
		}

		searcher := receipt.NewSearcher(gmailClient.Service(), gmailClient.UserID(), *outputDir)

		receipts, err := searcher.Search(ctx, receipt.SearchParams{
			FromDate: fromDate,
			ToDate:   toDate,
			Vendors:  vendorList,
			Label:    *gmailLabel,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Gmail search failed for %s: %v\n", cp, err)
			continue
		}
		fmt.Printf("  Found %d receipt emails\n", len(receipts))
		for _, r := range receipts {
			allReceipts = append(allReceipts, receiptWithSearcher{Receipt: r, Searcher: searcher})
		}
	}

	// Deduplicate receipts by MessageID
	receiptsWithSearcher := deduplicateReceiptsWithSearcher(allReceipts)
	fmt.Printf("Total: %d unique receipt emails\n", len(receiptsWithSearcher))

	if len(receiptsWithSearcher) == 0 {
		fmt.Println("No receipts found in Gmail")
		return
	}

	// Display receipts found
	fmt.Println()
	fmt.Println("Receipt emails found:")
	for _, rs := range receiptsWithSearcher {
		r := rs.Receipt
		subject := r.Subject
		if len(subject) > 50 {
			subject = subject[:50] + "..."
		}
		hasAttachment := "html"
		if len(r.Attachments) > 0 {
			hasAttachment = "pdf"
		}
		fmt.Printf("  %s  ¥%s  [%s] %s\n", r.Date.Format("2006-01-02"), formatAmount(r.Amount), hasAttachment, subject)
	}
	fmt.Println()

	// Step 4: Match transactions to receipts
	fmt.Println()
	fmt.Println("Matching transactions to receipts...")
	type match struct {
		TransactionPair transactionPair
		Receipt         *receipt.Receipt
		Searcher        *receipt.Searcher
	}
	var matches []match
	matchedReceipts := make(map[string]bool)

	for _, pair := range transactionPairs {
		txn := pair.Transaction
		for _, rs := range receiptsWithSearcher {
			r := rs.Receipt
			if matchedReceipts[r.MessageID] {
				continue
			}

			// Match by date (within tolerance) and amount
			daysDiff := int(math.Abs(float64(txn.Date.Sub(r.Date).Hours() / 24)))
			if daysDiff <= *toleranceDays && txn.Amount == r.Amount {
				matches = append(matches, match{TransactionPair: pair, Receipt: r, Searcher: rs.Searcher})
				matchedReceipts[r.MessageID] = true
				fmt.Printf("  %s ¥%s -> %s (%s)\n",
					txn.Date.Format("2006-01-02"),
					formatAmount(txn.Amount),
					r.Vendor,
					r.Subject[:min(40, len(r.Subject))],
				)
				break
			}
		}
	}

	fmt.Println()
	fmt.Printf("Matched %d transactions to receipts\n", len(matches))

	if len(matches) == 0 {
		fmt.Println("No matches found")
		return
	}

	// Initialize account mapper for deal creation
	accountMapper := freee.NewAccountMapper()

	// Step 5: Download matched receipts
	if *dryRun {
		fmt.Println()
		fmt.Println("[DRY RUN] Would download:")
		for _, m := range matches {
			txn := m.TransactionPair.Transaction
			mapping := accountMapper.GetMappingForTransaction(txn)
			fmt.Printf("  %s ¥%s -> %s\n",
				txn.Date.Format("2006-01-02"),
				formatAmount(txn.Amount),
				m.Receipt.Subject,
			)
			if *createDeals {
				fmt.Printf("    -> Would create deal: %s (%d), Tax: %s (%d)\n",
					mapping.AccountItemName, mapping.AccountItemID,
					mapping.TaxCodeName, mapping.TaxCode,
				)
			}
		}
		return
	}

	fmt.Println()
	fmt.Println("Downloading receipts...")
	downloaded := 0
	for i, m := range matches {
		fmt.Printf("[%d/%d] ", i+1, len(matches))

		result, err := m.Searcher.Download(ctx, m.Receipt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		switch result.Status {
		case "downloaded":
			fmt.Printf("Downloaded: %s\n", result.FilePath)
			downloaded++
		case "skipped":
			fmt.Printf("Skipped (exists): %s\n", result.FilePath)
		case "error":
			fmt.Printf("Error: %s\n", result.Error)
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Step 6: Create deals in freee (optional)
	dealsCreated := 0
	if *createDeals {
		fmt.Println()
		fmt.Println("Creating deals in freee...")
		for i, m := range matches {
			wt := m.TransactionPair.WalletTxn
			mapping := accountMapper.GetMappingForWalletTransaction(wt)

			fmt.Printf("[%d/%d] Creating deal for %s ¥%s... ",
				i+1, len(matches),
				wt.Date,
				formatAmount(abs(wt.Amount)),
			)

			dealResp, err := freeeClient.CreateDealFromTransaction(wt, mapping.AccountItemID, mapping.TaxCode)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			fmt.Printf("Created deal ID: %d (%s)\n", dealResp.Deal.ID, mapping.AccountItemName)
			dealsCreated++
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Printf("Unregistered transactions: %d\n", len(transactionPairs))
	fmt.Printf("Matched to Gmail receipts: %d\n", len(matches))
	fmt.Printf("Downloaded: %d\n", downloaded)
	if *createDeals {
		fmt.Printf("Deals created: %d\n", dealsCreated)
	}
	fmt.Println()
	fmt.Printf("Receipts saved to: %s\n", *outputDir)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func printHelp() {
	fmt.Println(`Gmail Receipt Auto-Fetcher

Fetches receipts from Gmail based on freee unregistered transactions.

Usage: auto-fetch [options]

Authentication (choose one):
  Option A: Direct access token
    --freee-token TOKEN        freee access token (or FREEE_ACCESS_TOKEN)
    --freee-company ID         freee company ID (or FREEE_COMPANY_ID)

  Option B: OAuth2 with auto-refresh (recommended)
    --freee-client-id ID       freee OAuth client ID (or FREEE_CLIENT_ID)
    --freee-client-secret SEC  freee OAuth client secret (or FREEE_CLIENT_SECRET)
    --freee-token-path FILE    freee token file path (or FREEE_TOKEN_PATH)

Options:
  --freee-api URL        freee API URL (FREEE_API_URL, default: https://api.freee.co.jp)
  --gmail-api URL        Gmail API URL for emulator (GMAIL_API_URL)
  --label LABEL          Gmail label to search (GMAIL_LABEL, e.g., "領収書")
  --vendors LIST         Comma-separated vendor names (RECEIPT_VENDORS)
  --output DIR           Output directory (RECEIPT_OUTPUT_DIR, default: "./receipts")
  --credentials FILES    Comma-separated Gmail credentials (GMAIL_CREDENTIALS_PATH, default: "credentials.json")
  --token-dir DIR        Directory for Gmail tokens (GMAIL_TOKEN_DIR, default: ".")
  --tolerance DAYS       Date matching tolerance (RECEIPT_TOLERANCE_DAYS, default: 3)
  --dry-run              Preview without downloading
  --create-deals         Create deals in freee for matched transactions
  -h, --help             Show this help

Environment Variables:
  FREEE_API_URL           freee API URL
  FREEE_ACCESS_TOKEN      freee access token
  FREEE_COMPANY_ID        freee company ID
  FREEE_CLIENT_ID         freee OAuth client ID
  FREEE_CLIENT_SECRET     freee OAuth client secret
  FREEE_TOKEN_PATH        freee token file path
  GMAIL_API_URL           Gmail API URL (for emulator)
  GMAIL_LABEL             Gmail label to search
  GMAIL_CREDENTIALS_PATH  Gmail OAuth credentials file paths (comma-separated)
  GMAIL_TOKEN_DIR         Directory for Gmail OAuth tokens
  RECEIPT_VENDORS         Comma-separated vendor names
  RECEIPT_OUTPUT_DIR      Output directory for downloaded PDFs
  RECEIPT_TOLERANCE_DAYS  Date matching tolerance in days

Examples:
  # Using OAuth2 auto-refresh (recommended)
  auto-fetch --freee-client-id YOUR_ID --freee-client-secret YOUR_SECRET

  # Using direct access token
  auto-fetch --freee-token TOKEN --freee-company 12345678

  # Preview matches without downloading
  auto-fetch --dry-run

  # Using multiple Gmail accounts
  auto-fetch --credentials "creds-work.json,creds-personal.json" --token-dir ./tokens`)
}

func parseVendors(s string) []string {
	if s == "" {
		return nil
	}
	var vendors []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			v := trim(s[start:i])
			if v != "" {
				vendors = append(vendors, v)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		v := trim(s[start:])
		if v != "" {
			vendors = append(vendors, v)
		}
	}
	return vendors
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func formatAmount(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// envOrDefault returns the flag value if set, otherwise the env var, otherwise the default.
func envOrDefault(flagVal, envKey, defaultVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultVal
}

// envOrDefaultInt returns the env var as int, or the default value.
func envOrDefaultInt(envKey string, defaultVal int) int {
	if v := os.Getenv(envKey); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

// parseCredentialsPaths parses comma-separated credentials paths.
func parseCredentialsPaths(s string) []string {
	if s == "" {
		return []string{"credentials.json"}
	}
	var paths []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			p := trim(s[start:i])
			if p != "" {
				paths = append(paths, p)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		p := trim(s[start:])
		if p != "" {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return []string{"credentials.json"}
	}
	return paths
}

// tokenFilename generates a token filename based on credentials path.
// e.g., "credentials.json" -> "token.json", "credentials-personal.json" -> "token-personal.json"
func tokenFilename(credPath string) string {
	base := filepath.Base(credPath)
	// Remove "credentials" prefix and get suffix
	if len(base) > len("credentials") && base[:len("credentials")] == "credentials" {
		suffix := base[len("credentials"):]
		return "token" + suffix
	}
	// Fallback: just use the same base name with token prefix
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	return "token-" + name + ext
}

// deduplicateReceiptsWithSearcher removes duplicate receipts by MessageID.
func deduplicateReceiptsWithSearcher(receipts []receiptWithSearcher) []receiptWithSearcher {
	seen := make(map[string]bool)
	var result []receiptWithSearcher
	for _, r := range receipts {
		if !seen[r.Receipt.MessageID] {
			seen[r.Receipt.MessageID] = true
			result = append(result, r)
		}
	}
	return result
}
