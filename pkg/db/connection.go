package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Connection manages a SQLite database connection.
type Connection struct {
	db     *sql.DB
	dbPath string
}

// Open opens a SQLite database connection.
// It enables WAL mode for better concurrency and foreign key constraints.
func Open(dbPath string) (*Connection, error) {
	// Ensure database file's parent directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database with SQLite driver
	// Connection string enables foreign keys and WAL mode
	connStr := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", dbPath)
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	conn := &Connection{
		db:     db,
		dbPath: dbPath,
	}

	// Initialize schema
	if err := InitializeSchema(conn); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return conn, nil
}

// Close closes the database connection.
func (c *Connection) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// GetDB returns the underlying *sql.DB instance.
// Use this for custom queries not covered by other methods.
func (c *Connection) GetDB() *sql.DB {
	return c.db
}

// GetPath returns the database file path.
func (c *Connection) GetPath() string {
	return c.dbPath
}

// Query executes a query that returns rows.
func (c *Connection) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.Query(query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
func (c *Connection) QueryRow(query string, args ...interface{}) *sql.Row {
	return c.db.QueryRow(query, args...)
}

// Exec executes a query that doesn't return rows.
// Returns sql.Result with information about the operation (LastInsertId, RowsAffected).
func (c *Connection) Exec(query string, args ...interface{}) (sql.Result, error) {
	return c.db.Exec(query, args...)
}

// Begin starts a new transaction.
func (c *Connection) Begin() (*sql.Tx, error) {
	return c.db.Begin()
}

// Transaction executes a function within a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (c *Connection) Transaction(fn func(*sql.Tx) error) error {
	tx, err := c.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
