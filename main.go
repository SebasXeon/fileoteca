package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"SebasXeon/Fileoteca/internal/addfile"
	"SebasXeon/Fileoteca/internal/ocr"
	"SebasXeon/Fileoteca/internal/shell"

	_ "SebasXeon/Fileoteca/migrations"

	"github.com/pocketbase/pocketbase/core"
)

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
	execPath, err := os.Executable()
	var ocrServerDir string
	if err == nil {
		execDir := filepath.Dir(execPath)
		ocrServerDir = filepath.Join(execDir, "ocr-server")
	} else {
		ocrServerDir = "ocr-server"
	}
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
					})
					go func() {
						time.Sleep(5 * time.Minute)
						cleanup()
					}()
				}()
				return e.Next()
			})

			log.Println("OCR integrado correctamente")
		}
	}

	fmt.Println("Fileoteca iniciada. Haz clic en el icono del área de notificación.")
	shell.StartTray(stopFn)
}
