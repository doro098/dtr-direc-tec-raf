package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Job struct {
	ID           int64     `json:"id"`
	JobID        string    `json:"job_id"`
	ItemID       string    `json:"item_id"`
	Source       string    `json:"source"`
	Text         string    `json:"text"`
	ExistingTags []string  `json:"existing_tags"`
	Status       string    `json:"status"`
	ResultTags   []string  `json:"result_tags,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Attempts     int       `json:"attempts"`
	MaxAttempts  int       `json:"max_attempts"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ProcessingAt *time.Time `json:"processing_at,omitempty"`
}

func (s *SQLite) InsertJob(job *Job) error {
	existingJSON := "[]"
	if len(job.ExistingTags) > 0 {
		b, _ := json.Marshal(job.ExistingTags)
		existingJSON = string(b)
	}

	_, err := s.db.Exec(
		`INSERT INTO jobs (job_id, item_id, source, text, existing_tags, status, max_attempts)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		job.JobID, job.ItemID, job.Source, job.Text, existingJSON, job.Status, job.MaxAttempts,
	)
	return err
}

func (s *SQLite) ClaimNextJob() (*Job, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRow(`
		SELECT id, job_id, item_id, source, text, existing_tags, status, attempts, max_attempts, created_at
		FROM jobs
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`)

	job := &Job{}
	var existingJSON string
	if err := row.Scan(&job.ID, &job.JobID, &job.ItemID, &job.Source, &job.Text,
		&existingJSON, &job.Status, &job.Attempts, &job.MaxAttempts, &job.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if existingJSON != "" {
		json.Unmarshal([]byte(existingJSON), &job.ExistingTags)
	}

	now := time.Now().UTC()
	_, err = tx.Exec(
		`UPDATE jobs SET status = 'processing', processing_at = ?, updated_at = ? WHERE id = ?`,
		now, now, job.ID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	job.Status = "processing"
	job.ProcessingAt = &now
	return job, nil
}

func (s *SQLite) CompleteJob(jobID string, tags []string) error {
	tagsJSON, _ := json.Marshal(tags)
	now := time.Now().UTC()
	_, err := s.db.Exec(
		`UPDATE jobs SET status = 'completed', result_tags = ?, updated_at = ? WHERE job_id = ?`,
		string(tagsJSON), now, jobID,
	)
	return err
}

func (s *SQLite) FailJob(jobID string, errMsg string) error {
	now := time.Now().UTC()
	_, err := s.db.Exec(
		`UPDATE jobs SET status = 'failed', error_message = ?, updated_at = ? WHERE job_id = ?`,
		errMsg, now, jobID,
	)
	return err
}

func (s *SQLite) IncrementAttempts(jobID string) error {
	now := time.Now().UTC()
	_, err := s.db.Exec(
		`UPDATE jobs SET attempts = attempts + 1, status = 'pending', processing_at = NULL, updated_at = ? WHERE job_id = ?`,
		now, jobID,
	)
	return err
}

func (s *SQLite) RecoverOrphans(timeout time.Duration) (int, error) {
	since := time.Now().UTC().Add(-timeout)
	res, err := s.db.Exec(
		`UPDATE jobs SET status = 'pending', processing_at = NULL, updated_at = CURRENT_TIMESTAMP
		 WHERE status = 'processing' AND processing_at < ?`,
		since,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *SQLite) GetJobByID(jobID string) (*Job, error) {
	row := s.db.QueryRow(`SELECT id, job_id, item_id, source, text, status, attempts, max_attempts, created_at FROM jobs WHERE job_id = ?`, jobID)
	job := &Job{}
	if err := row.Scan(&job.ID, &job.JobID, &job.ItemID, &job.Source, &job.Text,
		&job.Status, &job.Attempts, &job.MaxAttempts, &job.CreatedAt); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *SQLite) GetCategoryEmbedding(source, category string) ([]float64, error) {
	var jsonStr string
	err := s.db.QueryRow(
		`SELECT embedding FROM category_embeddings WHERE source = ? AND category = ?`,
		source, category,
	).Scan(&jsonStr)
	if err != nil {
		return nil, err
	}
	var emb []float64
	if err := json.Unmarshal([]byte(jsonStr), &emb); err != nil {
		return nil, err
	}
	return emb, nil
}

func (s *SQLite) SaveCategoryEmbedding(source, category string, embedding []float64) error {
	b, _ := json.Marshal(embedding)
	_, err := s.db.Exec(
		`INSERT INTO category_embeddings (source, category, embedding) VALUES (?, ?, ?)
		 ON CONFLICT(source, category) DO UPDATE SET embedding = excluded.embedding`,
		source, category, string(b),
	)
	return err
}

func (s *SQLite) GetJobsPending() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE status = 'pending'`).Scan(&n)
	return n, err
}

func (s *SQLite) GetJobsProcessing() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE status = 'processing'`).Scan(&n)
	return n, err
}
