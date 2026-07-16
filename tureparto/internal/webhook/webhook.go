package webhook

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/doro098/tureparto/internal/database"
)

type Handler struct {
	db *database.DB
}

type callbackPayload struct {
	JobID  string   `json:"job_id"`
	ItemID string   `json:"item_id"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload callbackPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("webhook: decode error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	log.Printf("webhook: received callback for item %s (job %s, status %s, tags: %v)",
		payload.ItemID, payload.JobID, payload.Status, payload.Tags)

	if payload.Status == "completed" {
		tagsJSON, _ := json.Marshal(payload.Tags)
		if err := h.db.UpdateCompleted(payload.ItemID, string(tagsJSON), payload.JobID); err != nil {
			log.Printf("webhook: error updating completion for %s: %v", payload.ItemID, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Always respond 200 OK to tag-mule
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
