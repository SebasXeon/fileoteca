package classifier

import (
	"log"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"
)

// RetrainQueue manages pending retrain requests with smart retry logic.
// It ensures we don't retrain a subcategory before its documents have OCR text.
type RetrainQueue struct {
	app       *pocketbase.PocketBase
	mgr       *ClassifierManager
	mu        sync.Mutex
	pending   map[string]time.Time // subcategoryID -> when it was first queued
	ticker    *time.Ticker
	quit      chan struct{}
	maxRetries int
	retryDelay time.Duration
}

func NewRetrainQueue(app *pocketbase.PocketBase, mgr *ClassifierManager) *RetrainQueue {
	return &RetrainQueue{
		app:        app,
		mgr:        mgr,
		pending:    make(map[string]time.Time),
		quit:       make(chan struct{}),
		maxRetries: 12, // 12 * 5s = 60s max wait
		retryDelay: 5 * time.Second,
	}
}

// Enqueue schedules a retrain for the given subcategory.
// If already pending, it resets the timer so it gets another retry window.
func (q *RetrainQueue) Enqueue(subcategoryID string) {
	if subcategoryID == "" {
		return
	}
	q.mu.Lock()
	q.pending[subcategoryID] = time.Now()
	q.mu.Unlock()
	log.Printf("classifier: retrain queued for %s", subcategoryID)
}

// Start begins the background worker that processes pending retrains.
func (q *RetrainQueue) Start() {
	q.ticker = time.NewTicker(q.retryDelay)
	go func() {
		for {
			select {
			case <-q.ticker.C:
				q.processPending()
			case <-q.quit:
				q.ticker.Stop()
				return
			}
		}
	}()
}

// Stop shuts down the queue worker.
func (q *RetrainQueue) Stop() {
	close(q.quit)
}

func (q *RetrainQueue) processPending() {
	q.mu.Lock()
	now := time.Now()
	toProcess := make(map[string]bool)
	for sid, t := range q.pending {
		// Only process items that have been pending for at least retryDelay
		// This gives OCR a chance to complete after the user assigns.
		if now.Sub(t) >= q.retryDelay {
			toProcess[sid] = true
		}
	}
	q.mu.Unlock()

	for sid := range toProcess {
		// Double-check: does this subcategory have documents with OCR text?
		docs, err := q.app.FindRecordsByFilter("documents",
			"subcategory_id = {:sid} && ocr_txt != ''", "", -1, 0,
			map[string]any{"sid": sid})
		if err != nil {
			log.Printf("classifier: retrain queue error querying docs for %s: %v", sid, err)
			continue
		}
		if len(docs) == 0 {
			// Still no OCR text. Keep it pending if within max retries window.
			q.mu.Lock()
			firstQueued, ok := q.pending[sid]
			q.mu.Unlock()
			if ok && now.Sub(firstQueued) > time.Duration(q.maxRetries)*q.retryDelay {
				log.Printf("classifier: retrain queue giving up on %s (no OCR after %d retries)", sid, q.maxRetries)
				q.mu.Lock()
				delete(q.pending, sid)
				q.mu.Unlock()
			}
			continue
		}

		// We have OCR text, proceed with retrain.
		if err := q.mgr.Retrain(sid); err != nil {
			log.Printf("classifier: retrain queue error for %s: %v", sid, err)
		}
		q.mu.Lock()
		delete(q.pending, sid)
		q.mu.Unlock()
	}
}

// OnDocumentOCRComplete should be called when a document finishes OCR processing.
// If the document already has a non-default subcategory assigned, we retrain that subcategory.
func (q *RetrainQueue) OnDocumentOCRComplete(docID string, defaultSubcategoryID string) {
	doc, err := q.app.FindRecordById("documents", docID)
	if err != nil {
		return
	}
	subID := doc.GetString("subcategory_id")
	if subID == "" || subID == defaultSubcategoryID {
		return
	}
	q.Enqueue(subID)
}
