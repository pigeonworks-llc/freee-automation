import Database from 'better-sqlite3';
import path from 'path';
import fs from 'fs';

/**
 * SQLite database connection manager for sync history
 *
 * Purpose:
 * - Manages connection to sync-history.db
 * - Provides query helpers
 * - Ensures WAL mode for better concurrency
 */
export class DatabaseConnection {
  private db: Database.Database;
  private dbPath: string;

  constructor(dbPath: string) {
    this.dbPath = path.resolve(dbPath);

    // Ensure database file exists
    if (!fs.existsSync(this.dbPath)) {
      throw new Error(
        `Database file not found: ${this.dbPath}\n` +
        'Please run: ./scripts/init-db.sh'
      );
    }

    // Open database with better-sqlite3
    this.db = new Database(this.dbPath, {
      verbose: process.env.DEBUG_SQL ? console.log : undefined,
    });

    // Enable WAL mode for better concurrency
    // WAL (Write-Ahead Logging) allows multiple readers while writing
    this.db.pragma('journal_mode = WAL');

    // Enable foreign key constraints
    this.db.pragma('foreign_keys = ON');
  }

  /**
   * Get the underlying database instance
   * Use this for custom queries not covered by sync-history.ts
   */
  getDb(): Database.Database {
    return this.db;
  }

  /**
   * Execute a query and return all results
   */
  query<T = any>(sql: string, params?: any[]): T[] {
    const stmt = this.db.prepare(sql);
    return (params && params.length > 0 ? stmt.all(params) : stmt.all()) as T[];
  }

  /**
   * Execute a query and return the first result
   */
  queryOne<T = any>(sql: string, params?: any[]): T | undefined {
    const stmt = this.db.prepare(sql);
    return (params && params.length > 0 ? stmt.get(params) : stmt.get()) as T | undefined;
  }

  /**
   * Execute an INSERT/UPDATE/DELETE query
   * Returns info about the operation (changes, lastInsertRowid)
   */
  execute(sql: string, params?: any[]): Database.RunResult {
    const stmt = this.db.prepare(sql);
    return params && params.length > 0 ? stmt.run(params) : stmt.run();
  }

  /**
   * Begin a transaction
   * Returns a transaction object with commit/rollback methods
   */
  transaction<T>(fn: () => T): T {
    return this.db.transaction(fn)();
  }

  /**
   * Close the database connection
   */
  close(): void {
    this.db.close();
  }

  /**
   * Check if database is open
   */
  isOpen(): boolean {
    return this.db.open;
  }

  /**
   * Get database path
   */
  getPath(): string {
    return this.dbPath;
  }
}

/**
 * Create a database connection with default path
 */
export function createConnection(dbPath?: string): DatabaseConnection {
  const defaultPath = path.join(process.cwd(), 'data', 'sync-history.db');
  return new DatabaseConnection(dbPath || defaultPath);
}

/**
 * Singleton instance for the default database
 */
let defaultConnection: DatabaseConnection | null = null;

/**
 * Get the default database connection
 * Creates one if it doesn't exist
 */
export function getDefaultConnection(): DatabaseConnection {
  if (!defaultConnection) {
    defaultConnection = createConnection();
  }
  return defaultConnection;
}

/**
 * Close the default database connection
 */
export function closeDefaultConnection(): void {
  if (defaultConnection) {
    defaultConnection.close();
    defaultConnection = null;
  }
}
