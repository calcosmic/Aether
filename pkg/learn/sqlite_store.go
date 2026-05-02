package learn

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteColonyStore implements LearnStore with SQLite persistence (HIVE-04).
// Drop-in replacement for ColonyStore (JSON) implementing the same LearnStore interface.
type SQLiteColonyStore struct {
	db       *sql.DB
	basePath string
	nextID   atomic.Int64
}

// NewSQLiteColonyStore opens (or creates) a SQLite database at dbPath in WAL mode.
// Runs all pending migrations on first open (D-02).
func NewSQLiteColonyStore(dbPath string) (*SQLiteColonyStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("learn: create db directory: %w", err)
	}

	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("learn: open sqlite: %w", err)
	}

	// Single writer constraint for SQLite
	db.SetMaxOpenConns(1)

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("learn: run migrations: %w", err)
	}

	return &SQLiteColonyStore{db: db, basePath: filepath.Dir(dbPath)}, nil
}

// Close closes the database connection.
func (s *SQLiteColonyStore) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection for use by SkillService and Curator.
func (s *SQLiteColonyStore) DB() *sql.DB {
	return s.db
}

// generateID creates a unique ID for a learning entry.
func (s *SQLiteColonyStore) generateID() string {
	seq := s.nextID.Add(1)
	return fmt.Sprintf("lrn_%s_%d", time.Now().Format("20060102"), seq)
}

// Add inserts a learning entry into the memories table.
func (s *SQLiteColonyStore) Add(entry Entry) error {
	if entry.ID == "" {
		entry.ID = s.generateID()
	}
	if entry.CreatedAt == "" {
		entry.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	evidenceJSON, err := json.Marshal(entry.Evidence)
	if err != nil {
		return fmt.Errorf("learn: marshal evidence: %w", err)
	}

	_, err = s.db.Exec(`INSERT INTO memories (id, content, evidence, classification, created_at, phase, caste, file_path, confidence, redacted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Content, string(evidenceJSON), string(entry.Classification),
		entry.CreatedAt, entry.Phase, entry.Caste, entry.FilePath,
		entry.Confidence, boolToInt(entry.Redacted))
	if err != nil {
		return fmt.Errorf("learn: add entry: %w", err)
	}
	return nil
}

// Get retrieves an entry by ID. Returns nil if not found.
func (s *SQLiteColonyStore) Get(id string) (*Entry, error) {
	row := s.db.QueryRow(`SELECT id, content, evidence, classification, created_at, phase, caste, file_path, confidence, redacted
		FROM memories WHERE id = ?`, id)
	entry, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("learn: get entry: %w", err)
	}
	return entry, nil
}

// List returns entries matching the filter.
func (s *SQLiteColonyStore) List(filter EntryFilter) ([]Entry, error) {
	query := `SELECT id, content, evidence, classification, created_at, phase, caste, file_path, confidence, redacted FROM memories WHERE 1=1`
	var args []interface{}

	if filter.Phase != 0 {
		query += ` AND phase = ?`
		args = append(args, filter.Phase)
	}
	if filter.Classification != "" {
		query += ` AND classification = ?`
		args = append(args, string(filter.Classification))
	}
	if filter.MinConfidence > 0 {
		query += ` AND confidence >= ?`
		args = append(args, filter.MinConfidence)
	}
	query += ` ORDER BY created_at DESC`

	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("learn: list entries: %w", err)
	}
	defer rows.Close()

	var result []Entry
	for rows.Next() {
		entry, err := scanEntryFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("learn: scan entry: %w", err)
		}
		result = append(result, *entry)
	}
	if result == nil {
		result = []Entry{}
	}
	return result, nil
}

// Replace updates an existing entry by ID.
func (s *SQLiteColonyStore) Replace(id string, entry Entry) error {
	evidenceJSON, err := json.Marshal(entry.Evidence)
	if err != nil {
		return fmt.Errorf("learn: marshal evidence: %w", err)
	}

	res, err := s.db.Exec(`UPDATE memories SET content = ?, evidence = ?, classification = ?,
		phase = ?, caste = ?, file_path = ?, confidence = ?, redacted = ? WHERE id = ?`,
		entry.Content, string(evidenceJSON), string(entry.Classification),
		entry.Phase, entry.Caste, entry.FilePath, entry.Confidence, boolToInt(entry.Redacted), id)
	if err != nil {
		return fmt.Errorf("learn: replace entry: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("learn: entry %q not found", id)
	}
	return nil
}

// Remove deletes an entry by ID.
func (s *SQLiteColonyStore) Remove(id string) error {
	res, err := s.db.Exec(`DELETE FROM memories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("learn: remove entry: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("learn: entry %q not found", id)
	}
	return nil
}

// Compact removes lowest-confidence entries until total content fits budget.
// Implemented in Task 1b -- stub returns nil.
func (s *SQLiteColonyStore) Compact(budget int) error {
	return nil
}

// MigrateFromJSON reads entries from the Phase 90 JSON format and inserts into SQLite.
// Implemented in Task 1b -- stub returns nil.
func (s *SQLiteColonyStore) MigrateFromJSON(jsonDir string) (int, error) {
	return 0, nil
}

// DBPath returns the database file path (for tests and CLI commands).
func (s *SQLiteColonyStore) DBPath() string {
	return filepath.Join(s.basePath, "colony.db")
}

// scanEntry scans a single entry from a sql.Row.
func scanEntry(row *sql.Row) (*Entry, error) {
	var e Entry
	var evidenceJSON string
	var redacted int
	err := row.Scan(&e.ID, &e.Content, &evidenceJSON, &e.Classification,
		&e.CreatedAt, &e.Phase, &e.Caste, &e.FilePath, &e.Confidence, &redacted)
	if err != nil {
		return nil, err
	}
	e.Redacted = redacted == 1
	if err := json.Unmarshal([]byte(evidenceJSON), &e.Evidence); err != nil {
		e.Evidence = Evidence{}
	}
	return &e, nil
}

// scanEntryFromRows scans a single entry from sql.Rows.
func scanEntryFromRows(rows *sql.Rows) (*Entry, error) {
	var e Entry
	var evidenceJSON string
	var redacted int
	err := rows.Scan(&e.ID, &e.Content, &evidenceJSON, &e.Classification,
		&e.CreatedAt, &e.Phase, &e.Caste, &e.FilePath, &e.Confidence, &redacted)
	if err != nil {
		return nil, err
	}
	e.Redacted = redacted == 1
	if err := json.Unmarshal([]byte(evidenceJSON), &e.Evidence); err != nil {
		e.Evidence = Evidence{}
	}
	return &e, nil
}

// boolToInt converts bool to SQLite integer (0/1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
