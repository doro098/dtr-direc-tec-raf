package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/doro098/tag_mule/internal/config"
	"github.com/doro098/tag_mule/internal/storage"
	"github.com/doro098/tag_mule/internal/worker"
	"github.com/google/uuid"
)

type Handler struct {
	db   *storage.SQLite
	pool *worker.Pool
	cfg  *config.Config
}

func NewHandler(db *storage.SQLite, pool *worker.Pool, cfg *config.Config) *Handler {
	return &Handler{db: db, pool: pool, cfg: cfg}
}

type enrichRequest struct {
	Source       string   `json:"source"`
	ItemID       string   `json:"item_id"`
	Text         string   `json:"text"`
	ExistingTags []string `json:"existing_tags"`
}

type enrichResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func (h *Handler) Enrich(w http.ResponseWriter, r *http.Request) {
	var req enrichRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid JSON body",
			Code:  "INVALID_JSON",
		})
		return
	}

	if req.Source == "" || req.ItemID == "" || req.Text == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "missing required field: source, item_id, text",
			Code:  "MISSING_FIELD",
		})
		return
	}

	if _, ok := h.cfg.AppConfig.Sources[req.Source]; !ok {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "unknown source: " + req.Source,
			Code:  "INVALID_SOURCE",
		})
		return
	}

	if req.ExistingTags == nil {
		req.ExistingTags = []string{}
	}

	jobID := uuid.New().String()
	job := &storage.Job{
		JobID:        jobID,
		ItemID:       req.ItemID,
		Source:       req.Source,
		Text:         req.Text,
		ExistingTags: req.ExistingTags,
		Status:       "pending",
		MaxAttempts:  h.cfg.MaxAttempts,
	}

	if err := h.db.InsertJob(job); err != nil {
		log.Printf("insert job error: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "internal error",
			Code:  "INTERNAL",
		})
		return
	}

	h.pool.Notify()

	writeJSON(w, http.StatusOK, enrichResponse{
		JobID:  jobID,
		Status: "queued",
	})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	pending, _ := h.db.GetJobsPending()
	processing, _ := h.db.GetJobsProcessing()

	dbConnected := true
	_, err := h.db.DB().Exec("SELECT 1")
	if err != nil {
		dbConnected = false
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"workers_active": h.cfg.WorkerCount,
		"jobs_pending":   pending + processing,
		"db_connected":   dbConnected,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
