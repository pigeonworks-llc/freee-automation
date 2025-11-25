package beancount

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shunichi-ikebuchi/accounting-system/pkg/pathutil"
)

// Repository defines the interface for Beancount file operations.
type Repository interface {
	// AppendTransaction appends a transaction to a monthly file
	AppendTransaction(yearMonth, transaction string, comment ...string) error

	// ReadMonthFile reads the content of a monthly file
	ReadMonthFile(yearMonth string) (string, error)

	// MonthFileExists checks if a monthly file exists
	MonthFileExists(yearMonth string) bool

	// GetMonthFilesInYear gets all monthly files in a year
	GetMonthFilesInYear(year string) ([]string, error)

	// EnsureMonthFile ensures a monthly file exists with header
	EnsureMonthFile(yearMonth string) error
}

// FileSystemRepository is a file system implementation of Repository.
type FileSystemRepository struct {
	pathResolver *pathutil.PathResolver
}

// NewFileSystemRepository creates a new FileSystemRepository.
func NewFileSystemRepository(pathResolver *pathutil.PathResolver) *FileSystemRepository {
	return &FileSystemRepository{
		pathResolver: pathResolver,
	}
}

// AppendTransaction appends a transaction to a monthly file.
// It creates the file if it doesn't exist.
func (r *FileSystemRepository) AppendTransaction(yearMonth, transaction string, comment ...string) error {
	filePath, err := r.pathResolver.GetMonthFilePath(yearMonth)
	if err != nil {
		return fmt.Errorf("failed to get month file path: %w", err)
	}

	// Ensure file exists with header
	if err := r.EnsureMonthFile(yearMonth); err != nil {
		return fmt.Errorf("failed to ensure month file: %w", err)
	}

	// Prepare content to append
	var content string
	if len(comment) > 0 && comment[0] != "" {
		content += fmt.Sprintf("; %s\n", comment[0])
	}
	content += transaction
	if len(transaction) > 0 && transaction[len(transaction)-1] != '\n' {
		content += "\n"
	}
	content += "\n" // Add blank line after transaction

	// Append to file
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for appending: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// ReadMonthFile reads the content of a monthly file.
// Returns empty string if file doesn't exist.
func (r *FileSystemRepository) ReadMonthFile(yearMonth string) (string, error) {
	filePath, err := r.pathResolver.GetMonthFilePath(yearMonth)
	if err != nil {
		return "", fmt.Errorf("failed to get month file path: %w", err)
	}

	if !r.pathResolver.FileExists(filePath) {
		return "", nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}

// MonthFileExists checks if a monthly file exists.
func (r *FileSystemRepository) MonthFileExists(yearMonth string) bool {
	filePath, err := r.pathResolver.GetMonthFilePath(yearMonth)
	if err != nil {
		return false
	}

	return r.pathResolver.FileExists(filePath)
}

// GetMonthFilesInYear gets all monthly files in a year.
// Returns a slice of year-month strings (e.g., ["2024-01", "2024-02"]).
func (r *FileSystemRepository) GetMonthFilesInYear(year string) ([]string, error) {
	yearDir := r.pathResolver.GetYearDir(year)
	if !r.pathResolver.FileExists(yearDir) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(yearDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read year directory: %w", err)
	}

	var monthFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) == ".beancount" {
			// Remove .beancount extension to get YYYY-MM
			monthKey := name[:len(name)-len(".beancount")]
			monthFiles = append(monthFiles, monthKey)
		}
	}

	return monthFiles, nil
}

// EnsureMonthFile ensures a monthly file exists with header.
// If the file already exists, this is a no-op.
func (r *FileSystemRepository) EnsureMonthFile(yearMonth string) error {
	filePath, err := r.pathResolver.GetMonthFilePath(yearMonth)
	if err != nil {
		return fmt.Errorf("failed to get month file path: %w", err)
	}

	if r.pathResolver.FileExists(filePath) {
		return nil
	}

	// Ensure parent directory exists
	if err := r.pathResolver.EnsureParentDir(filePath); err != nil {
		return fmt.Errorf("failed to ensure parent directory: %w", err)
	}

	// Create file with header
	header := r.generateFileHeader(yearMonth)
	if err := os.WriteFile(filePath, []byte(header), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// generateFileHeader generates a header comment for a monthly file.
func (r *FileSystemRepository) generateFileHeader(yearMonth string) string {
	now := time.Now().Format(time.RFC3339)
	return fmt.Sprintf("; Beancount file for %s\n; Generated at %s\n\n", yearMonth, now)
}
