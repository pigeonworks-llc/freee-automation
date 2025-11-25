// Package main provides CLI for Gmail receipt fetcher.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shunichi-ikebuchi/accounting-system/gmail-receipt-fetcher/internal/gmail"
	"github.com/shunichi-ikebuchi/accounting-system/gmail-receipt-fetcher/internal/receipt"
)

func main() {
	// Parse flags
	fromDate := flag.String("from", "", "Start date (YYYY-MM-DD)")
	toDate := flag.String("to", "", "End date (YYYY-MM-DD)")
	vendors := flag.String("vendors", "amazon", "Comma-separated vendor names to search")
	outputDir := flag.String("output", "./receipts", "Output directory for downloaded PDFs")
	credPath := flag.String("credentials", "credentials.json", "Path to Google OAuth credentials")
	tokenPath := flag.String("token", "", "Path to store OAuth token (default: ~/.config/gmail-fetcher/token.json)")
	dryRun := flag.Bool("dry-run", false, "Preview without downloading")
	help := flag.Bool("help", false, "Show help")
	flag.BoolVar(help, "h", false, "Show help")

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Validate dates
	var from, to time.Time
	var err error

	if *fromDate != "" {
		from, err = time.Parse("2006-01-02", *fromDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid from date: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Default: 30 days ago
		from = time.Now().AddDate(0, 0, -30)
	}

	if *toDate != "" {
		to, err = time.Parse("2006-01-02", *toDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid to date: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Default: today
		to = time.Now()
	}

	// Set default token path
	if *tokenPath == "" {
		home, _ := os.UserHomeDir()
		*tokenPath = filepath.Join(home, ".config", "gmail-fetcher", "token.json")
	}

	// Parse vendors
	vendorList := parseVendors(*vendors)

	fmt.Println("=== Gmail Receipt Fetcher ===")
	fmt.Println()
	fmt.Printf("Date range: %s to %s\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
	fmt.Printf("Vendors: %v\n", vendorList)
	fmt.Printf("Output: %s\n", *outputDir)
	if *dryRun {
		fmt.Println("Mode: DRY RUN (no downloads)")
	}
	fmt.Println()

	// Check credentials file
	if _, err := os.Stat(*credPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: credentials file not found: %s\n", *credPath)
		fmt.Fprintf(os.Stderr, "\nTo set up Gmail API credentials:\n")
		fmt.Fprintf(os.Stderr, "1. Go to https://console.cloud.google.com/apis/credentials\n")
		fmt.Fprintf(os.Stderr, "2. Create OAuth 2.0 Client ID (Desktop app)\n")
		fmt.Fprintf(os.Stderr, "3. Download credentials.json\n")
		fmt.Fprintf(os.Stderr, "4. Place it in the current directory or specify with --credentials\n")
		os.Exit(1)
	}

	// Initialize Gmail client
	ctx := context.Background()
	client, err := gmail.NewClient(ctx, gmail.Config{
		CredentialsPath: *credPath,
		TokenPath:       *tokenPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create Gmail client: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Gmail client initialized successfully")
	fmt.Println()

	// Create searcher
	searcher := receipt.NewSearcher(client.Service(), client.UserID(), *outputDir)

	// Search for receipts
	fmt.Println("Searching for receipt emails...")
	receipts, err := searcher.Search(ctx, receipt.SearchParams{
		FromDate: from,
		ToDate:   to,
		Vendors:  vendorList,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: search failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d receipt emails with PDF attachments\n", len(receipts))
	fmt.Println()

	if len(receipts) == 0 {
		fmt.Println("No receipts to download")
		return
	}

	// List receipts
	fmt.Println("Receipts found:")
	for i, r := range receipts {
		fmt.Printf("  [%d] %s  %s  %s\n", i+1, r.Date.Format("2006-01-02"), r.Vendor, r.Subject)
		for _, a := range r.Attachments {
			fmt.Printf("      Attachment: %s (%d bytes)\n", a.Filename, a.Size)
		}
	}
	fmt.Println()

	// Download
	if *dryRun {
		fmt.Println("[DRY RUN] Would download the above receipts")
		return
	}

	fmt.Println("Downloading receipts...")
	downloaded := 0
	skipped := 0
	errors := 0

	for i, r := range receipts {
		fmt.Printf("[%d/%d] %s %s: ", i+1, len(receipts), r.Date.Format("2006-01-02"), r.Vendor)

		result, err := searcher.Download(ctx, r)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			errors++
			continue
		}

		switch result.Status {
		case "downloaded":
			fmt.Printf("Downloaded: %s\n", result.FilePath)
			downloaded++
		case "skipped":
			fmt.Printf("Skipped (already exists)\n")
			skipped++
		case "error":
			fmt.Printf("Error: %s\n", result.Error)
			errors++
		}

		// Rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	// Summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Printf("Total found: %d\n", len(receipts))
	fmt.Printf("Downloaded: %d\n", downloaded)
	fmt.Printf("Skipped: %d\n", skipped)
	fmt.Printf("Errors: %d\n", errors)
	fmt.Println()
	fmt.Printf("Receipt PDFs saved to: %s\n", *outputDir)
}

func printHelp() {
	fmt.Println(`Gmail Receipt Fetcher

Searches Gmail for receipt emails and downloads PDF attachments.

Usage: gmail-fetcher [options]

Options:
  --from DATE          Start date (YYYY-MM-DD, default: 30 days ago)
  --to DATE            End date (YYYY-MM-DD, default: today)
  --vendors LIST       Comma-separated vendor names (default: "amazon")
  --output DIR         Output directory (default: "./receipts")
  --credentials FILE   Path to Google OAuth credentials (default: "credentials.json")
  --token FILE         Path to store OAuth token
  --dry-run            Preview without downloading
  -h, --help           Show this help

Examples:
  # Search Amazon receipts from last 30 days
  gmail-fetcher --vendors amazon

  # Search multiple vendors with date range
  gmail-fetcher --from 2024-01-01 --to 2024-12-31 --vendors "amazon,rakuten,yodobashi"

  # Preview without downloading
  gmail-fetcher --dry-run

Setup:
  1. Go to https://console.cloud.google.com/apis/credentials
  2. Create OAuth 2.0 Client ID (Desktop app)
  3. Enable Gmail API for your project
  4. Download credentials.json
  5. Run gmail-fetcher (will prompt for OAuth)`)
}

func parseVendors(s string) []string {
	if s == "" {
		return nil
	}
	var vendors []string
	for _, v := range splitTrim(s, ",") {
		if v != "" {
			vendors = append(vendors, v)
		}
	}
	return vendors
}

func splitTrim(s, sep string) []string {
	var result []string
	for _, part := range split(s, sep) {
		trimmed := trim(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
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
