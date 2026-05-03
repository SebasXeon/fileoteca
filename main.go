package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"SebasXeon/Fileoteca/internal/addfile"
	"SebasXeon/Fileoteca/internal/classifier"
	"SebasXeon/Fileoteca/internal/ocr"
	"SebasXeon/Fileoteca/internal/shell"

	_ "SebasXeon/Fileoteca/migrations"

	"github.com/pocketbase/pocketbase/core"
)

func resolveOcrServerDir() string {
	// Try relative to working directory first (for dev: `go run .`)
	cwd, _ := os.Getwd()
	cwdCandidate := filepath.Join(cwd, "ocr-server")
	if info, err := os.Stat(cwdCandidate); err == nil && info.IsDir() {
		return cwdCandidate
	}

	// Try relative to executable (for production: Fileoteca.exe next to ocr-server/)
	exec, err := os.Executable()
	if err == nil {
		exeCandidate := filepath.Join(filepath.Dir(exec), "ocr-server")
		if info, err := os.Stat(exeCandidate); err == nil && info.IsDir() {
			return exeCandidate
		}
	}

	return "ocr-server"
}

func main() {
	var addFilePath string

	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--add" && i+1 < len(os.Args) {
			addFilePath = os.Args[i+1]
			break
		}
	}

	if addFilePath != "" {
		info, err := addfile.Extract(addFilePath)
		if err != nil {
			shell.ShowError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

		if shell.IsServerRunning() {
			if err := shell.AddFileViaHTTP(info); err != nil {
				shell.ShowError(fmt.Sprintf("Error agregando documento:\n%v", err))
				os.Exit(1)
			}
			shell.ShowInfo(fmt.Sprintf("\"%s\" agregado a Fileoteca.", info.FileName))
			return
		}

		newArgs := os.Args[:1]
		for i := 1; i < len(os.Args); i++ {
			if os.Args[i] == "--add" {
				i++
				continue
			}
			newArgs = append(newArgs, os.Args[i])
		}
		os.Args = newArgs

		app, stopFn, err := shell.StartServer()
		if err != nil {
			log.Fatalf("error iniciando servidor: %v\n", err)
		}

		if err := shell.AddFileViaDAO(app, info); err != nil {
			log.Printf("error agregando documento vía DAO: %v\n", err)
		}

		shell.StartTray(stopFn)
		return
	}

	app, stopFn, err := shell.StartServer()
	if err != nil {
		log.Fatalf("error iniciando servidor: %v\n", err)
	}

	// Start OCR server
	ocrServerDir := resolveOcrServerDir()
	ocrServer, ocrErr := ocr.StartOcrServer(ocrServerDir)
	if ocrErr != nil {
		log.Printf("aviso: OCR server no disponible: %v", ocrErr)
	} else {
		defer ocrServer.Stop()

		ocrClient, clientErr := ocr.NewOcrClient(ocr.OcrServerAddr())
		if clientErr != nil {
			log.Printf("aviso: cliente OCR no disponible: %v", clientErr)
		} else {
			defer ocrClient.Close()
			ocrWorker := ocr.NewOcrWorker(ocrClient, app, 100)
			ocrWorker.Start()
			defer ocrWorker.Stop()

			classifierMgr := classifier.NewClassifierManager(app, "pb_data/models")
			classifierMgr.Load()

			app.OnRecordCreate("documents").BindFunc(func(e *core.RecordEvent) error {
				go func() {
					resolvedPath, cleanup, err := ocr.ResolvePath(e.Record)
					if err != nil {
						log.Printf("OCR skip para %s: %v", e.Record.Id, err)
						return
					}
					ocrWorker.Enqueue(ocr.OcrJob{
						ID:       e.Record.Id,
						FilePath: resolvedPath,
						FileType: e.Record.GetString("file_ext"),
						OnComplete: func(ocrText string) {
							classifierMgr.ClassifyAndAssign(e.Record.Id, ocrText)
						},
					})
					go func() {
						time.Sleep(5 * time.Minute)
						cleanup()
					}()
				}()
				return e.Next()
			})

			app.OnRecordUpdate("documents").BindFunc(func(e *core.RecordEvent) error {
				go func() {
					oldSub := e.Record.Original().GetString("subcategory_id")
					newSub := e.Record.GetString("subcategory_id")
					if oldSub != newSub && newSub != "" {
						cfg, _ := shell.LoadConfig()
						if newSub == cfg.DefaultSubcategoryID {
							return
						}
						if err := classifierMgr.Retrain(newSub); err != nil {
							log.Printf("classifier: retrain error: %v", err)
						}
					}
				}()
				return e.Next()
			})

			log.Println("OCR integrado correctamente")
		}
	}

	fmt.Println("Fileoteca iniciada. Haz clic en el icono del área de notificación.")
	shell.StartTray(stopFn)
}
