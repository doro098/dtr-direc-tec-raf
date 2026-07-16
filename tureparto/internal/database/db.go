package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	SourceDB *sql.DB // Read-Only: tureparto.db (original WhatsApp data)
	RichDB   *sql.DB // Read-Write: tureparto_rich.db (AI-enriched data)
}

type RawMessage struct {
	ID           int64  `json:"id"`
	FromNumber   string `json:"from_number"`
	MessageBody  string `json:"message_body"`
	ReceivedAt   string `json:"received_at"`
}

type AIEnrichment struct {
	OriginalMsgID string `json:"original_msg_id"`
	PhoneNumber   string `json:"phone_number"`
	MessageBody   string `json:"message_body"`
	Status        string `json:"status"`
	TagMuleJobID  string `json:"tag_mule_job_id,omitempty"`
	SuggestedTags string `json:"suggested_tags,omitempty"`
	ErrorDetails  string `json:"error_details,omitempty"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func Open(sourcePath, richPath string) (*DB, error) {
	sourceDB, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", sourcePath))
	if err != nil {
		return nil, fmt.Errorf("open source db: %w", err)
	}

	richDB, err := sql.Open("sqlite3", richPath)
	if err != nil {
		sourceDB.Close()
		return nil, fmt.Errorf("open rich db: %w", err)
	}

	if err := migrateRich(richDB); err != nil {
		sourceDB.Close()
		richDB.Close()
		return nil, fmt.Errorf("migrate rich db: %w", err)
	}

	return &DB{
		SourceDB: sourceDB,
		RichDB:   richDB,
	}, nil
}

func migrateRich(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS ai_enrichment (
		original_msg_id TEXT PRIMARY KEY,
		phone_number TEXT NOT NULL,
		message_body TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		tag_mule_job_id TEXT,
		suggested_tags TEXT,
		error_details TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_ai_status ON ai_enrichment(status);`
	_, err := db.Exec(schema)
	return err
}

func (d *DB) Close() {
	d.SourceDB.Close()
	d.RichDB.Close()
}

func (d *DB) GetRecentMessages(limit int) ([]RawMessage, error) {
	rows, err := d.SourceDB.Query(
		`SELECT id, from_number, message_body, received_at FROM messages ORDER BY received_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []RawMessage
	for rows.Next() {
		var m RawMessage
		if err := rows.Scan(&m.ID, &m.FromNumber, &m.MessageBody, &m.ReceivedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (d *DB) ExistsInRich(msgID int64) (bool, error) {
	var count int
	err := d.RichDB.QueryRow(
		`SELECT COUNT(*) FROM ai_enrichment WHERE original_msg_id = ?`,
		fmt.Sprintf("%d", msgID),
	).Scan(&count)
	return count > 0, err
}

func (d *DB) InsertPending(msgID int64, phone, body string) error {
	_, err := d.RichDB.Exec(
		`INSERT INTO ai_enrichment (original_msg_id, phone_number, message_body, status)
		 VALUES (?, ?, ?, 'pending')`,
		fmt.Sprintf("%d", msgID), phone, body,
	)
	return err
}

func (d *DB) UpdateSentToMule(msgID string, jobID string) error {
	_, err := d.RichDB.Exec(
		`UPDATE ai_enrichment SET status = 'sent_to_mule', tag_mule_job_id = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE original_msg_id = ?`,
		jobID, msgID,
	)
	return err
}

func (d *DB) UpdateCompleted(msgID string, tags string, jobID string) error {
	_, err := d.RichDB.Exec(
		`UPDATE ai_enrichment SET status = 'completed', suggested_tags = ?, tag_mule_job_id = COALESCE(NULLIF(tag_mule_job_id, ''), ?), updated_at = CURRENT_TIMESTAMP
		 WHERE original_msg_id = ?`,
		tags, jobID, msgID,
	)
	return err
}

func (d *DB) GetRecentEnrichments(limit int) ([]AIEnrichment, error) {
	rows, err := d.RichDB.Query(
		`SELECT original_msg_id, phone_number, message_body, status,
		        COALESCE(tag_mule_job_id, ''), COALESCE(suggested_tags, ''),
		        COALESCE(error_details, ''), created_at, updated_at
		 FROM ai_enrichment ORDER BY updated_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AIEnrichment
	for rows.Next() {
		var item AIEnrichment
		if err := rows.Scan(&item.OriginalMsgID, &item.PhoneNumber, &item.MessageBody,
			&item.Status, &item.TagMuleJobID, &item.SuggestedTags,
			&item.ErrorDetails, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
