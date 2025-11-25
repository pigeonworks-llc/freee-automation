# Archived TypeScript Packages

This directory contains archived TypeScript implementations that have been replaced by Go versions.

## Contents

### freee-sync-ts/
Original TypeScript implementation of freee-sync.
**Replaced by:** `cmd/freee-sync/` (Go version)

### shared-ts/
Original TypeScript shared library (config, pathutil, beancount repository).
**Replaced by:** `pkg/` (Go packages)

## Note

These packages are kept for reference only.
Do not use them in new development - use the Go versions instead.

## Go Equivalents

| Archived (TypeScript) | Active (Go) |
|----------------------|-------------|
| `shared-ts/src/config/` | `pkg/config/` |
| `shared-ts/src/utils/path-resolver.ts` | `pkg/pathutil/` |
| `shared-ts/src/repository/` | `pkg/beancount/` |
| `freee-sync-ts/src/freee/` | `pkg/freee/` |
| `freee-sync-ts/src/converter/` | `pkg/converter/` |
| `freee-sync-ts/src/db/` | `pkg/db/` |
| `freee-sync-ts/src/sync.ts` | `cmd/freee-sync/cmd/sync.go` |
