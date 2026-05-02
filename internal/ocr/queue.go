package ocr

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type OcrJob struct {
	ID       string
	FilePath string
	FileType string
}

type OcrWorker struct {
	client *OcrClient
	jobs   chan OcrJob
	app    *pocketbase.PocketBase
	quit   chan struct{}
}

func NewOcrWorker(client *OcrClient, app *pocketbase.PocketBase, bufferSize int) *OcrWorker {
	return &OcrWorker{
		client: client,
		jobs:   make(chan OcrJob, bufferSize),
		app:    app,
		quit:   make(chan struct{}),
	}
}

func (w *OcrWorker) Enqueue(job OcrJob) {
	select {
	case w.jobs <- job:
	default:
		log.Printf("⚠ OCR queue full, dropping job for document %s", job.ID)
	}
}

func (w *OcrWorker) Start() {
	go func() {
		for {
			select {
			case job := <-w.jobs:
				w.processJob(job)
			case <-w.quit:
				return
			}
		}
	}()
	log.Println("OCR worker started")
}

func (w *OcrWorker) Stop() {
	close(w.quit)
}

func (w *OcrWorker) processJob(job OcrJob) {
	log.Printf("OCR processing document %s (%s)", job.ID, job.FileType)

	ctx := context.Background()
	text, err := w.client.ExtractText(ctx, job.ID, job.FilePath, job.FileType)

	if err != nil {
		log.Printf("OCR error for document %s: %v", job.ID, err)
		w.updateDocumentStatus(job.ID, "error", "")
		return
	}

	w.updateDocumentStatus(job.ID, "processed", text)
	log.Printf("OCR complete for document %s (%d chars)", job.ID, len(text))
}

func (w *OcrWorker) updateDocumentStatus(id string, status string, ocrText string) {
	err := w.app.RunInTransaction(func(txApp core.App) error {
		record, err := txApp.FindRecordById("documents", id)
		if err != nil {
			return fmt.Errorf("record not found %s: %w", id, err)
		}
		record.Set("status", status)
		record.Set("ocr_txt", ocrText)
		return txApp.Save(record)
	})
	if err != nil {
		log.Printf("Failed to update document %s: %v", id, err)
	}
}

// ResolvePath returns a readable file path for a document record.
func ResolvePath(record *core.Record) (string, func(), error) {
	path := record.GetString("path")
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, func() {}, nil
		}
	}

	fileName := record.GetString("file")
	if fileName == "" {
		return "", nil, fmt.Errorf("document %s has no file path or uploaded file", record.Id)
	}

	col := record.Collection()

	srcPath := filepath.Join("pb_data", "storage", col.Id, record.Id, fileName)
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open PB storage file for %s: %w", record.Id, err)
	}
	defer srcFile.Close()

	ext := filepath.Ext(fileName)
	tmpDir := filepath.Join(os.TempDir(), "fileoteca")
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	tmpPath := filepath.Join(tmpDir, record.Id+ext)
	dstFile, err := os.Create(tmpPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", nil, fmt.Errorf("failed to copy to temp: %w", err)
	}

	cleanup := func() {
		os.Remove(tmpPath)
	}

	return tmpPath, cleanup, nil
}
