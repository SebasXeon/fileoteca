package main

import (
	"fmt"
	"log"
	"os"

	"SebasXeon/Fileoteca/internal/addfile"
	"SebasXeon/Fileoteca/internal/shell"

	_ "SebasXeon/Fileoteca/migrations"
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

	_, stopFn, err := shell.StartServer()
	if err != nil {
		log.Fatalf("error iniciando servidor: %v\n", err)
	}

	fmt.Println("Fileoteca iniciada. Haz clic en el icono del área de notificación.")
	shell.StartTray(stopFn)
}
