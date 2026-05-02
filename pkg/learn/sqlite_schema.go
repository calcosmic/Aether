package learn

// SQL statements for colony.db schema (D-01: single database, all tables).
// Tables: schema_version, memories, runs, workers, gates, skills, decisions, trajectories.
const (
	sqlCreateSchemaVersion = `CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`

	sqlCreateMemories = `CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		evidence TEXT NOT NULL DEFAULT '{}',
		classification TEXT NOT NULL DEFAULT 'repo-local',
		created_at TEXT NOT NULL,
		phase INTEGER NOT NULL DEFAULT 0,
		caste TEXT NOT NULL DEFAULT '',
		file_path TEXT NOT NULL DEFAULT '',
		confidence REAL NOT NULL DEFAULT 0,
		redacted INTEGER NOT NULL DEFAULT 0
	)`

	sqlCreateMemoriesIdx = `CREATE INDEX IF NOT EXISTS idx_memories_phase ON memories(phase)`
	sqlCreateMemoriesIdx2 = `CREATE INDEX IF NOT EXISTS idx_memories_classification ON memories(classification)`
	sqlCreateMemoriesIdx3 = `CREATE INDEX IF NOT EXISTS idx_memories_confidence ON memories(confidence)`

	sqlCreateRuns = `CREATE TABLE IF NOT EXISTS runs (
		id TEXT PRIMARY KEY,
		phase INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		started_at TEXT NOT NULL,
		completed_at TEXT NOT NULL DEFAULT '',
		worker_count INTEGER NOT NULL DEFAULT 0,
		gates_passed INTEGER NOT NULL DEFAULT 0,
		gates_total INTEGER NOT NULL DEFAULT 0
	)`

	sqlCreateWorkers = `CREATE TABLE IF NOT EXISTS workers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id TEXT NOT NULL,
		name TEXT NOT NULL,
		caste TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		files_touched TEXT NOT NULL DEFAULT '[]',
		started_at TEXT NOT NULL DEFAULT '',
		completed_at TEXT NOT NULL DEFAULT '',
		FOREIGN KEY (run_id) REFERENCES runs(id)
	)`

	sqlCreateGates = `CREATE TABLE IF NOT EXISTS gates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id TEXT NOT NULL,
		gate_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'not-reached',
		message TEXT NOT NULL DEFAULT '',
		checked_at TEXT NOT NULL DEFAULT '',
		FOREIGN KEY (run_id) REFERENCES runs(id)
	)`

	sqlCreateSkills = `CREATE TABLE IF NOT EXISTS skills (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		stage TEXT NOT NULL DEFAULT 'active',
		pinned INTEGER NOT NULL DEFAULT 0,
		view_count INTEGER NOT NULL DEFAULT 0,
		use_count INTEGER NOT NULL DEFAULT 0,
		patch_count INTEGER NOT NULL DEFAULT 0,
		last_used_at TEXT,
		last_viewed_at TEXT,
		created_at TEXT NOT NULL,
		last_transitioned_at TEXT NOT NULL DEFAULT '',
		source_run_id TEXT NOT NULL DEFAULT '',
		confidence REAL NOT NULL DEFAULT 0,
		auto_created INTEGER NOT NULL DEFAULT 0,
		file_path TEXT NOT NULL DEFAULT ''
	)`

	sqlCreateSkillsIdx = `CREATE INDEX IF NOT EXISTS idx_skills_stage ON skills(stage)`
	sqlCreateSkillsIdx2 = `CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name)`

	sqlCreateDecisions = `CREATE TABLE IF NOT EXISTS decisions (
		id TEXT PRIMARY KEY,
		phase INTEGER NOT NULL,
		content TEXT NOT NULL,
		rationale TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	)`

	sqlCreateTrajectories = `CREATE TABLE IF NOT EXISTS trajectories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id TEXT NOT NULL,
		phase INTEGER NOT NULL,
		from_state TEXT NOT NULL,
		to_state TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		FOREIGN KEY (run_id) REFERENCES runs(id)
	)`

	// FTS5 external content virtual table for memories (D-03: unified FTS5 index)
	sqlCreateMemoriesFTS = `CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
		content,
		category,
		content=memories,
		content_rowid=rowid
	)`

	// Sync triggers to keep FTS index in sync with memories table
	sqlTriggerMemoriesAI = `CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
		INSERT INTO memories_fts(rowid, content, category) VALUES (new.rowid, new.content, new.classification);
	END`

	sqlTriggerMemoriesAD = `CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, content, category) VALUES('delete', old.rowid, old.content, old.classification);
	END`

	sqlTriggerMemoriesAU = `CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, content, category) VALUES('delete', old.rowid, old.content, old.classification);
		INSERT INTO memories_fts(rowid, content, category) VALUES (new.rowid, new.content, new.classification);
	END`
)
