package visor

import (
	"encoding/json"
	"net/http"

	"github.com/doro098/tureparto/internal/database"
)

type Visor struct {
	db *database.DB
}

func New(db *database.DB) *Visor {
	return &Visor{db: db}
}

func (v *Visor) DataHandler(w http.ResponseWriter, r *http.Request) {
	items, err := v.db.GetRecentEnrichments(50)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
