package shell

import (
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"
	"golang.org/x/sys/windows"
)

var mutexHandle windows.Handle

func createMutex() error {
	name, _ := syscall.UTF16PtrFromString("FileotecaServer")
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		return fmt.Errorf("error creando mutex: %w", err)
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(handle)
		return fmt.Errorf("otra instancia de Fileoteca ya está corriendo")
	}
	mutexHandle = handle
	return nil
}

func releaseMutex() {
	if mutexHandle != 0 {
		windows.CloseHandle(mutexHandle)
		mutexHandle = 0
	}
}

func IsServerRunning() bool {
	name, _ := syscall.UTF16PtrFromString("FileotecaServer")
	handle, err := windows.OpenMutex(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return false
	}
	windows.CloseHandle(handle)
	return true
}

func ensureDefaultCategory(app *pocketbase.PocketBase) error {
	categories, err := app.FindCollectionByNameOrId("categories")
	if err != nil {
		return fmt.Errorf("no se encontró la colección categories: %w", err)
	}

	subcategories, err := app.FindCollectionByNameOrId("subcategories")
	if err != nil {
		return fmt.Errorf("no se encontró la colección subcategories: %w", err)
	}

	catRecords, err := app.FindRecordsByFilter("categories", "name = {:name}", "", -1, 0,
		map[string]any{"name": "Sin categorizar"})
	if err != nil {
		return fmt.Errorf("error buscando categoría por defecto: %w", err)
	}

	var catID string
	if len(catRecords) == 0 {
		rec := core.NewRecord(categories)
		rec.Set("name", "Sin categorizar")
		rec.Set("description", "Categoría por defecto para archivos agregados")
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("error creando categoría por defecto: %w", err)
		}
		catID = rec.Id
	} else {
		catID = catRecords[0].Id
	}

	subRecords, err := app.FindRecordsByFilter("subcategories",
		"name = {:name} && category_id = {:cat_id}", "", -1, 0,
		map[string]any{"name": "General", "cat_id": catID})
	if err != nil {
		return fmt.Errorf("error buscando subcategoría por defecto: %w", err)
	}

	if len(subRecords) == 0 {
		rec := core.NewRecord(subcategories)
		rec.Set("name", "General")
		rec.Set("category_id", catID)
		rec.Set("model_name", "general")
		rec.Set("is_default", true)
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("error creando subcategoría por defecto: %w", err)
		}
	}

	return nil
}

func StartServer() (*pocketbase.PocketBase, func(), error) {
	if err := createMutex(); err != nil {
		return nil, nil, err
	}

	installed, err := EnsureContextMenu()
	if err != nil {
		log.Printf("aviso: no se pudo registrar el menú contextual: %v", err)
	} else if installed {
		log.Println("menú contextual registrado correctamente")
	}

	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: osutils.IsProbablyGoRun(),
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/{path...}", apis.Static(os.DirFS("./pb_public"), false))
		return se.Next()
	})

	bootstrapDone := make(chan struct{})
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		close(bootstrapDone)
		return e.Next()
	})

	go func() {
		if err := app.Start(); err != nil {
			log.Fatalf("error iniciando servidor: %v", err)
		}
	}()

	<-bootstrapDone

	if err := ensureDefaultCategory(app); err != nil {
		log.Printf("aviso: no se pudo crear categoría por defecto: %v", err)
	}

	stopFn := func() {
		log.Println("deteniendo servidor...")
		releaseMutex()
	}

	return app, stopFn, nil
}
