package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/doro098/tag_mule/internal/config"
	"github.com/doro098/tag_mule/internal/providers"
	"github.com/doro098/tag_mule/internal/storage"
)

type Pool struct {
	db       *storage.SQLite
	cfg      *config.Config
	count    int
	stopCh   chan struct{}
	notifyCh chan struct{}
	wg       sync.WaitGroup
}

func NewPool(db *storage.SQLite, cfg *config.Config, count int) *Pool {
	return &Pool{
		db:       db,
		cfg:      cfg,
		count:    count,
		stopCh:   make(chan struct{}),
		notifyCh: make(chan struct{}, 100),
	}
}

func (p *Pool) Notify() {
	select {
	case p.notifyCh <- struct{}{}:
	default:
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.count; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.wg.Add(1)
	go p.orphanWatcher()
	log.Printf("started %d workers + orphan watcher", p.count)
}

func (p *Pool) Shutdown() {
	close(p.stopCh)
	p.wg.Wait()
	log.Println("all workers stopped")
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()

	ollamaProvider := providers.NewOllama(p.cfg.OllamaURL)
	var zaiProvider providers.Provider
	if p.cfg.ZaiAPIKey != "" {
		zaiProvider = providers.NewZAI(p.cfg.ZaiBaseURL, p.cfg.ZaiAPIKey)
	}

	for {
		select {
		case <-p.stopCh:
			log.Printf("worker %d stopping", id)
			return
		case <-p.notifyCh:
			p.processJobs(id, ollamaProvider, zaiProvider)
		case <-time.After(5 * time.Second):
			p.processJobs(id, ollamaProvider, zaiProvider)
		}
	}
}

func (p *Pool) processJobs(id int, ollamaProvider, zaiProvider providers.Provider) {
	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		job, err := p.db.ClaimNextJob()
		if err != nil {
			log.Printf("worker %d: claim error: %v", id, err)
			return
		}
		if job == nil {
			return
		}

		log.Printf("worker %d: processing job %s (source: %s)", id, job.JobID, job.Source)
		p.processJob(job, ollamaProvider, zaiProvider)
	}
}

func (p *Pool) processJob(job *storage.Job, ollamaProvider, zaiProvider providers.Provider) {
	sourceCfg, ok := p.cfg.AppConfig.Sources[job.Source]
	if !ok {
		p.db.FailJob(job.JobID, "source config not found")
		return
	}

	var tagger providers.Tagger

	if sourceCfg.Method == "llm" {
		tagger = providers.NewLLMTagger(ollamaProvider)
	} else if sourceCfg.Method == "embedding" {
		tagger = providers.NewEmbeddingTagger(ollamaProvider, p.db, job.Source)
	} else {
		p.db.FailJob(job.JobID, "unknown method: "+sourceCfg.Method)
		return
	}

	tags, err := tagger.Tag(job.Text, job.ExistingTags, sourceCfg)
	if err != nil {
		log.Printf("job %s: tag error: %v", job.JobID, err)
		p.db.FailJob(job.JobID, err.Error())
		return
	}

	if err := p.db.CompleteJob(job.JobID, tags); err != nil {
		log.Printf("job %s: complete error: %v", job.JobID, err)
		return
	}

	p.callback(job, tags, sourceCfg)
}

func (p *Pool) callback(job *storage.Job, tags []string, sourceCfg config.SourceConfig) {
	payload := map[string]interface{}{
		"job_id":  job.JobID,
		"item_id": job.ItemID,
		"status":  "completed",
		"tags":    tags,
	}

	data, _ := json.Marshal(payload)

	for attempt := 0; attempt < p.cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Minute
			log.Printf("job %s: callback retry %d in %v", job.JobID, attempt, backoff)
			time.Sleep(backoff)
		}

		client := &http.Client{Timeout: p.cfg.CallbackTimeout}
		resp, err := client.Post(sourceCfg.CallbackURL, "application/json", bytes.NewReader(data))
		if err != nil {
			log.Printf("job %s: callback attempt %d error: %v", job.JobID, attempt, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("job %s: callback success", job.JobID)
			return
		}

		log.Printf("job %s: callback attempt %d got status %d", job.JobID, attempt, resp.StatusCode)
	}

	log.Printf("job %s: all callback attempts exhausted", job.JobID)
}

func (p *Pool) orphanWatcher() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cfg.OrphanCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			recovered, err := p.db.RecoverOrphans(p.cfg.OrphanTimeout)
			if err != nil {
				log.Printf("orphan check error: %v", err)
				continue
			}
			if recovered > 0 {
				log.Printf("recovered %d orphan jobs", recovered)
				p.Notify()
			}
		}
	}
}
