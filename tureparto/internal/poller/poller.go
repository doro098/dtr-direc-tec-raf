package poller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/doro098/tureparto/internal/database"
)

type Poller struct {
	db       *database.DB
	muleURL  string
	interval time.Duration
	stopCh   chan struct{}
}

func New(db *database.DB, muleURL string, intervalSeconds int) *Poller {
	return &Poller{
		db:       db,
		muleURL:  muleURL,
		interval: time.Duration(intervalSeconds) * time.Second,
		stopCh:   make(chan struct{}),
	}
}

func (p *Poller) Start() {
	log.Printf("poller started (interval: %v)", p.interval)
	go func() {
		// Run immediately on start, then on ticker
		p.poll()
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		for {
			select {
			case <-p.stopCh:
				log.Println("poller stopped")
				return
			case <-ticker.C:
				p.poll()
			}
		}
	}()
}

func (p *Poller) Stop() {
	close(p.stopCh)
}

func (p *Poller) poll() {
	messages, err := p.db.GetRecentMessages(50)
	if err != nil {
		log.Printf("poller: error reading messages: %v", err)
		return
	}

	for _, msg := range messages {
		exists, err := p.db.ExistsInRich(msg.ID)
		if err != nil {
			log.Printf("poller: error checking existence msg %d: %v", msg.ID, err)
			continue
		}
		if exists {
			continue
		}

		if err := p.db.InsertPending(msg.ID, msg.FromNumber, msg.MessageBody); err != nil {
			log.Printf("poller: error inserting msg %d: %v", msg.ID, err)
			continue
		}

		jobID, err := p.sendToTagMule(msg)
		if err != nil {
			log.Printf("poller: error sending msg %d to tag-mule: %v", msg.ID, err)
			continue
		}

		msgID := fmt.Sprintf("%d", msg.ID)
		if err := p.db.UpdateSentToMule(msgID, jobID); err != nil {
			log.Printf("poller: error updating status for msg %s: %v", msgID, err)
		}
	}
}

type enrichPayload struct {
	Source       string   `json:"source"`
	ItemID       string   `json:"item_id"`
	Text         string   `json:"text"`
	ExistingTags []string `json:"existing_tags"`
}

type enrichResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

func (p *Poller) sendToTagMule(msg database.RawMessage) (string, error) {
	payload := enrichPayload{
		Source:       "tureparto",
		ItemID:       fmt.Sprintf("%d", msg.ID),
		Text:         msg.MessageBody,
		ExistingTags: []string{},
	}

	data, _ := json.Marshal(payload)
	resp, err := http.Post(p.muleURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tag-mule returned status %d", resp.StatusCode)
	}

	var result enrichResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.JobID, nil
}
