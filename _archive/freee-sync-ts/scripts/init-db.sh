#!/bin/bash
# SQLite database initialization script
# Creates sync-history database with schema

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SCHEMA_FILE="$PROJECT_ROOT/schema/sync-database.sql"
DB_DIR="$PROJECT_ROOT/data"
DB_FILE="$DB_DIR/sync-history.db"

# Create data directory if it doesn't exist
mkdir -p "$DB_DIR"

# Check if schema file exists
if [ ! -f "$SCHEMA_FILE" ]; then
  echo "Error: Schema file not found at $SCHEMA_FILE"
  exit 1
fi

# Initialize database
echo "Initializing database at $DB_FILE..."
sqlite3 "$DB_FILE" < "$SCHEMA_FILE"

echo "✓ Database initialized successfully"
echo "  Location: $DB_FILE"
echo "  Schema: $SCHEMA_FILE"

# Show tables
echo ""
echo "Database tables:"
sqlite3 "$DB_FILE" ".tables"
