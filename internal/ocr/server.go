package ocr

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	defaultOcrPort       = "50051"
	ocrServerStartupWait = 3 * time.Second
)

type OcrServer struct {
	process *exec.Cmd
}

func StartOcrServer(ocrServerDir string) (*OcrServer, error) {
	fullPath, err := filepath.Abs(ocrServerDir)
	if err != nil {
		return nil, fmt.Errorf("invalid ocr server path: %w", err)
	}

	cmd := exec.Command("uv", "run", "--directory", fullPath, "ocr-server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start OCR server: %w", err)
	}

	log.Printf("OCR server started (PID %d)", cmd.Process.Pid)
	time.Sleep(ocrServerStartupWait)

	return &OcrServer{process: cmd}, nil
}

func (s *OcrServer) Stop() error {
	if s.process == nil || s.process.Process == nil {
		return nil
	}
	log.Println("stopping OCR server...")
	if err := s.process.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill OCR server: %w", err)
	}
	_, _ = s.process.Process.Wait()
	log.Println("OCR server stopped")
	return nil
}

func OcrServerAddr() string {
	return fmt.Sprintf("localhost:%s", defaultOcrPort)
}
