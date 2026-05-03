package shell

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"

	"SebasXeon/Fileoteca/internal/api"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"
	"github.com/pocketbase/pocketbase/tools/types"
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

	var subID string
	if len(subRecords) == 0 {
		rec := core.NewRecord(subcategories)
		rec.Set("name", "General")
		rec.Set("category_id", catID)
		rec.Set("model_name", "general")
		rec.Set("is_default", true)
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("error creando subcategoría por defecto: %w", err)
		}
		subID = rec.Id
	} else {
		subID = subRecords[0].Id
	}

	cfg, _ := LoadConfig()
	if cfg.DefaultCategoryID != catID || cfg.DefaultSubcategoryID != subID {
		cfg.DefaultCategoryID = catID
		cfg.DefaultSubcategoryID = subID
		_ = SaveConfig(cfg)
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

	ready := make(chan struct{})
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/documents/open/{id}", api.OpenDocumentHandler(app))

		staticHandler := apis.Static(os.DirFS("./pb_public"), false)
		se.Router.GET("/{path...}", func(e *core.RequestEvent) error {
			if strings.HasPrefix(e.Request.URL.Path, "/api/") {
				return e.Next()
			}
			if err := staticHandler(e); err != nil {
				http.ServeFile(e.Response, e.Request, "./pb_public/index.html")
				return nil
			}
			return nil
		})

		close(ready)
		return se.Next()
	})

	go func() {
		if err := app.Start(); err != nil {
			log.Fatalf("error iniciando servidor: %v", err)
		}
	}()

	<-ready

	if err := ensureDefaultCategory(app); err != nil {
		log.Printf("aviso: no se pudo crear categoría por defecto: %v", err)
	}

	// Ensure documents collection allows creation (migration sets it to read-only)
	docs, err := app.FindCollectionByNameOrId("documents")
	if err == nil && docs.CreateRule == nil {
		docs.CreateRule = types.Pointer("")
		if err := app.Save(docs); err != nil {
			log.Printf("aviso: no se pudo actualizar reglas de documents: %v", err)
		} else {
			log.Println("reglas de creación actualizadas para documents")
		}
	}

	stopFn := func() {
		log.Println("deteniendo servidor...")
		releaseMutex()
	}

	return app, stopFn, nil
}
