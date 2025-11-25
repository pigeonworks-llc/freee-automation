// Package pathutil provides centralized path management for Beancount files and directories.
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathResolver manages paths for Beancount files, database, and attachments.
type PathResolver struct {
	beancountRoot  string
	databasePath   string
	attachmentsDir string
}

// Config represents the configuration for PathResolver.
type Config struct {
	// BeancountRoot is the root directory for all Beancount files (e.g., ~/accounting/beancount)
	BeancountRoot string
	// DatabasePath is the path to the SQLite database file for sync history
	DatabasePath string
	// AttachmentsDir is the directory for receipts, invoices, etc.
	AttachmentsDir string
}

// New creates a new PathResolver with the given configuration.
// If DatabasePath is empty, it defaults to {BeancountRoot}/.sync/sync.db
// If AttachmentsDir is empty, it defaults to {BeancountRoot}/attachments
func New(config Config) *PathResolver {
	dbPath := config.DatabasePath
	if dbPath == "" {
		dbPath = filepath.Join(config.BeancountRoot, ".sync", "sync.db")
	}

	attachmentsDir := config.AttachmentsDir
	if attachmentsDir == "" {
		attachmentsDir = filepath.Join(config.BeancountRoot, "attachments")
	}

	return &PathResolver{
		beancountRoot:  config.BeancountRoot,
		databasePath:   dbPath,
		attachmentsDir: attachmentsDir,
	}
}

// FromEnv creates a PathResolver from environment variables.
// Expected environment variables:
//   - BEANCOUNT_ROOT: Root directory for Beancount files (required)
//   - BEANCOUNT_DB_PATH: Database file path (optional)
//   - BEANCOUNT_ATTACHMENTS_DIR: Attachments directory (optional)
func FromEnv() (*PathResolver, error) {
	beancountRoot := os.Getenv("BEANCOUNT_ROOT")
	if beancountRoot == "" {
		return nil, fmt.Errorf("BEANCOUNT_ROOT environment variable is required")
	}

	return New(Config{
		BeancountRoot:  beancountRoot,
		DatabasePath:   os.Getenv("BEANCOUNT_DB_PATH"),
		AttachmentsDir: os.Getenv("BEANCOUNT_ATTACHMENTS_DIR"),
	}), nil
}

// GetBeancountRoot returns the Beancount root directory.
func (p *PathResolver) GetBeancountRoot() string {
	return p.beancountRoot
}

// GetDatabasePath returns the database file path.
func (p *PathResolver) GetDatabasePath() string {
	return p.databasePath
}

// GetAttachmentsDir returns the attachments directory.
func (p *PathResolver) GetAttachmentsDir() string {
	return p.attachmentsDir
}

// GetYearDir returns the directory path for a year.
// Example: ~/accounting/beancount/2024
func (p *PathResolver) GetYearDir(year string) string {
	return filepath.Join(p.beancountRoot, year)
}

// GetMonthFilePath returns the file path for a month.
// yearMonth should be in YYYY-MM format.
// Example: ~/accounting/beancount/2024/2024-01.beancount
func (p *PathResolver) GetMonthFilePath(yearMonth string) (string, error) {
	parts := strings.Split(yearMonth, "-")
	if len(parts) != 2 || len(parts[0]) != 4 || len(parts[1]) != 2 {
		return "", fmt.Errorf("invalid year-month format: %s. Expected YYYY-MM", yearMonth)
	}

	year := parts[0]
	yearDir := p.GetYearDir(year)
	filename := fmt.Sprintf("%s.beancount", yearMonth)

	return filepath.Join(yearDir, filename), nil
}

// GetAttachmentPath returns the attachment file path for a given date and filename.
// It creates subdirectories by year/month.
// Example: attachments/2024/01/receipt.pdf
func (p *PathResolver) GetAttachmentPath(date, filename string) (string, error) {
	parts := strings.Split(date, "-")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid date format: %s. Expected YYYY-MM-DD", date)
	}

	year := parts[0]
	month := parts[1]

	return filepath.Join(p.attachmentsDir, year, month, filename), nil
}

// EnsureDir creates a directory if it doesn't exist.
// It creates all parent directories as needed (like mkdir -p).
func (p *PathResolver) EnsureDir(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}
	return nil
}

// EnsureParentDir ensures the parent directory of a file exists.
func (p *PathResolver) EnsureParentDir(filePath string) error {
	dir := filepath.Dir(filePath)
	return p.EnsureDir(dir)
}

// FileExists checks if a file exists.
func (p *PathResolver) FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// IsDir checks if a path is a directory.
func (p *PathResolver) IsDir(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}
