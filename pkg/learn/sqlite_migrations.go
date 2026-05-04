package learn

import (
	"database/sql"
	"fmt"
)

// migrations maps version numbers to migration functions (D-02: Go map, no third-party).
// Each function runs inside a transaction for atomicity.
// New migrations are appended -- NEVER modify existing migration functions.
var migrations = map[int]func(*sql.Tx) error{
	1: migrateV1_CreateTables,
	2: migrateV2_CreateIndexes,
	3: migrateV3_CreateFTS5,
}

// migrateV1_CreateTables creates all base tables (D-01: single colony.db).
func migrateV1_CreateTables(tx *sql.Tx) error {
	tables := []string{
		sqlCreateSchemaVersion,
		sqlCreateMemories,
		sqlCreateRuns,
		sqlCreateWorkers,
		sqlCreateGates,
		sqlCreateSkills,
		sqlCreateDecisions,
		sqlCreateTrajectories,
	}
	for _, ddl := range tables {
		if _, err := tx.Exec(ddl); err != nil {
			return fmt.Errorf("exec DDL: %w", err)
		}
	}
	return nil
}

// migrateV2_CreateIndexes adds performance indexes.
func migrateV2_CreateIndexes(tx *sql.Tx) error {
	indexes := []string{
		sqlCreateMemoriesIdx,
		sqlCreateMemoriesIdx2,
		sqlCreateMemoriesIdx3,
		sqlCreateSkillsIdx,
		sqlCreateSkillsIdx2,
	}
	for _, ddl := range indexes {
		if _, err := tx.Exec(ddl); err != nil {
			return fmt.Errorf("exec index DDL: %w", err)
		}
	}
	return nil
}

// migrateV3_CreateFTS5 creates the FTS5 virtual table and sync triggers (D-03).
// Must run AFTER v1 (memories table must exist for external content reference).
func migrateV3_CreateFTS5(tx *sql.Tx) error {
	stmts := []string{
		sqlCreateMemoriesFTS,
		sqlTriggerMemoriesAI,
		sqlTriggerMemoriesAD,
		sqlTriggerMemoriesAU,
	}
	for _, ddl := range stmts {
		if _, err := tx.Exec(ddl); err != nil {
			return fmt.Errorf("exec FTS5 DDL: %w", err)
		}
	}
	return nil
}

// runMigrations applies all pending migrations in version order.
// Idempotent: reads current version from schema_version, only runs newer migrations.
// Each migration runs in its own transaction.
func runMigrations(db *sql.DB) error {
	// Ensure schema_version table exists (handled by v1, but guard against empty DB)
	if _, err := db.Exec(sqlCreateSchemaVersion); err != nil {
		return fmt.Errorf("learn: ensure schema_version table: %w", err)
	}

	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&current); err != nil {
		return fmt.Errorf("learn: read schema version: %w", err)
	}

	for v := current + 1; v <= len(migrations); v++ {
		fn, ok := migrations[v]
		if !ok {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("learn: begin migration v%d: %w", v, err)
		}
		if err := fn(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("learn: migration v%d: %w", v, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES (?)`, v); err != nil {
			tx.Rollback()
			return fmt.Errorf("learn: record migration v%d: %w", v, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("learn: commit migration v%d: %w", v, err)
		}
	}
	return nil
}
