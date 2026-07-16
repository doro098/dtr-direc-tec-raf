package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	db *sql.DB
}

func NewSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	db.SetMaxOpenConns(1)

	return &SQLite{db: db}, nil
}

func (s *SQLite) DB() *sql.DB {
	return s.db
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT UNIQUE NOT NULL,
		item_id TEXT NOT NULL,
		source TEXT NOT NULL,
		text TEXT NOT NULL,
		existing_tags TEXT,
		status TEXT DEFAULT 'pending',
		result_tags TEXT,
		error_message TEXT,
		attempts INTEGER DEFAULT 0,
		max_attempts INTEGER DEFAULT 3,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		processing_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	CREATE INDEX IF NOT EXISTS idx_jobs_pending ON jobs(status, created_at);
	CREATE INDEX IF NOT EXISTS idx_jobs_orphan ON jobs(status, processing_at);

	CREATE TABLE IF NOT EXISTS category_embeddings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		category TEXT NOT NULL,
		embedding TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(source, category)
	);
	`
	_, err := s.db.Exec(schema)
	return err
}
