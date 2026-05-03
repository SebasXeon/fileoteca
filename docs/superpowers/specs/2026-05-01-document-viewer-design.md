# Diseño: Visualizador de Documentos

## Resumen
Agregar una función para visualizar documentos agregados a Fileoteca. El backend expone una ruta custom que decide si sirve el archivo para visualización en navegador o lo abre externamente. El frontend tiene una nueva página de detalle del documento.

## Alcance
- Ruta custom en PocketBase: `/api/open/:id`
- Página frontend: `/document/:id`
- Navegación clickeable desde listas de documentos
- Soporte para dos fuentes de archivo: `path` local o `file` subido a PocketBase

---

## Backend

### Ruta: `GET /api/open/:id`

1. Busca el registro en la colección `documents` por `id`.
2. Determina la fuente del archivo en este orden:
   - Si `path` existe y el archivo existe en disco → usa `path`.
   - Si no, pero tiene `file` subido a PocketBase → usa el archivo de storage de PocketBase.
   - Si ninguno → retorna HTTP 404.
3. Determina el tipo MIME por la extensión del archivo.
4. **Si el tipo es visualizable en navegador** (`image/*`, `application/pdf`, `text/*`, `application/json`, `application/xml`, `text/html`):
   - Responde con `Content-Type` correcto.
   - Responde con `Content-Disposition: inline`.
   - Sirve el archivo como stream de bytes (`http.ServeContent` o similar).
5. **Si el tipo NO es visualizable** (docx, xlsx, pptx, etc.):
   - En Windows, ejecuta `cmd /c start <path>` para abrir con la app por defecto del sistema.
   - Retorna JSON: `{ "action": "opened_externally" }`.
   - Si el archivo viene de PocketBase (`file`) y no de `path`, primero lo extrae a un temp y luego abre externamente.

### Tipos visualizables en navegador
- Imágenes: `png`, `jpg`, `jpeg`, `gif`, `bmp`, `svg`, `webp`, `tiff`, `ico`
- PDF: `pdf`
- Texto: `txt`, `csv`, `md`, `rtf`, `json`, `xml`, `html`, `htm`

---

## Frontend

### Nueva ruta: `/document/[id]/+page.svelte`

#### Carga de datos
- Obtiene el documento vía `pb.collection("documents").getOne(id, { expand: "category_id,subcategory_id" })`.
- Determina si es visualizable por la extensión (`file_ext`).

#### UI
- **Panel de información** (arriba o a la izquierda):
  - Nombre del archivo
  - Extensión
  - Tamaño (formateado)
  - Fecha de actualización
  - Categoría / Subcategoría
  - Notas (si existen)
  - Botón "Abrir externamente" (siempre disponible)
- **Panel de visualización**:
  - Si es **visualizable**: muestra el archivo en un contenedor apropiado:
    - PDF → `<iframe src="/api/open/{id}">`
    - Imagen → `<img src="/api/open/{id}">`
    - Texto → fetch a `/api/open/{id}` y muestra en `<pre>`
  - Si es **NO visualizable**: muestra mensaje:
    > "Abriendo documento en su aplicación predeterminada..."
    - Llama a `/api/open/{id}` y muestra confirmación cuando el backend responde `{ action: "opened_externally" }`.

### Navegación clickeable
- En `+page.svelte` (inicio): cada `FileCard` y cada fila de lista ahora son enlaces a `/document/{file.id}`.
- En `documents/+page.svelte`: los botones "Abrir" llevan a `/document/{id}`.

### API helpers
- `getDocument(id)` en `$lib/api/documents.ts`
- `openDocument(id)` en `$lib/api/documents.ts` (llama a `/api/open/:id`)

---

## Decisiones de diseño

### Fuente del archivo
Se prefiere el `path` local sobre el `file` de PocketBase, porque:
- Los documentos "agregados" vía menú contextual apuntan al archivo original del usuario.
- El `file` de PocketBase es fallback para documentos subidos manualmente desde el navegador.

### Apertura externa en Windows
Se usa `cmd /c start` porque:
- Es la forma estándar en Windows de abrir un archivo con su aplicación por defecto.
- No requiere conocer qué app maneja qué extensión.

### No se usan rutas de PocketBase nativas
- La ruta `/api/files/...` de PocketBase sirve archivos subidos, pero no maneja archivos locales por `path`.
- La ruta custom unifica ambas fuentes y añade la lógica de "abrir externamente".

---

## Archivos a modificar/crear

### Backend
- `internal/shell/server.go` — registrar la ruta custom en `OnServe()`
- Nuevo archivo: `internal/api/open.go` — handler de la ruta `/api/open/:id`

### Frontend
- Nuevo: `web/src/routes/document/[id]/+page.svelte`
- `web/src/lib/api/documents.ts` — agregar `getDocument(id)` y `openDocument(id)`
- `web/src/routes/+page.svelte` — hacer clickeables los documentos
- `web/src/routes/documents/+page.svelte` — enlazar botones "Abrir" a `/document/:id`
- `web/src/lib/app-nav.ts` — (opcional) añadir título para `/document/:id` en breadcrumb

---

## Consideraciones futuras
- En sistemas no-Windows, la apertura externa debería usar `xdg-open` o `open`.
- Si el archivo de PocketBase es grande, el streaming podría beneficiarse de rangos HTTP (no crítico para MVP).

