package learn

import (
	"fmt"
	"strings"
)

// Search performs full-text search across memory content using FTS5 (D-03, D-04).
// query is natural language text (e.g., "memory leak test failure").
// filter applies additional constraints (classification, confidence, limit).
// Returns entries ranked by BM25 relevance.
func (s *SQLiteColonyStore) Search(query string, filter EntryFilter) ([]Entry, error) {
	// Sanitize FTS5 query: escape special characters that could cause syntax errors
	sanitized := sanitizeFTS5Query(query)
	if sanitized == "" {
		return []Entry{}, nil
	}

	where := "memories_fts MATCH ?"
	args := []interface{}{sanitized}

	if filter.Classification != "" {
		where += " AND m.classification = ?"
		args = append(args, string(filter.Classification))
	}
	if filter.MinConfidence > 0 {
		where += " AND m.confidence >= ?"
		args = append(args, filter.MinConfidence)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	sqlQuery := fmt.Sprintf(`
		SELECT m.id, m.content, m.evidence, m.classification, m.created_at,
		       m.phase, m.caste, m.file_path, m.confidence, m.redacted
		FROM memories_fts f
		JOIN memories m ON m.rowid = f.rowid
		WHERE %s
		ORDER BY rank
		LIMIT ?`, where)

	args = append(args, limit)
	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("learn: search: %w", err)
	}
	defer rows.Close()

	var result []Entry
	for rows.Next() {
		entry, err := scanEntryFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("learn: search scan: %w", err)
		}
		result = append(result, *entry)
	}
	if result == nil {
		result = []Entry{}
	}
	return result, nil
}

// sanitizeFTS5Query escapes FTS5 special characters from user input.
// FTS5 special characters: AND, OR, NOT, *, ^, ", (, ), {, }, :, +
// Strategy: split on whitespace, quote each token, join with AND.
func sanitizeFTS5Query(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	tokens := strings.Fields(query)
	var sanitized []string
	for _, token := range tokens {
		// Skip FTS5 operators
		upper := strings.ToUpper(token)
		if upper == "AND" || upper == "OR" || upper == "NOT" {
			continue
		}
		// Strip FTS5 special characters
		token = strings.Trim(token, "*\"(){}:^+")
		if token == "" {
			continue
		}
		sanitized = append(sanitized, `"`+token+`"`)
	}
	return strings.Join(sanitized, " AND ")
}
