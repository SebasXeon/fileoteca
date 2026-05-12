# Thumbnail Generation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add thumbnail generation to the document-tools server (OCR server) — extract first page of PDFs and resize images to 400px-wide JPEGs, store in PocketBase `thumbnail` file field, display in frontend.

**Architecture:** New `GenerateThumbnail` gRPC method on the Python server generates a thumbnail image to a temp file and returns the path. The Go OCR worker calls it after OCR completes, reads the temp file, uploads it to PocketBase's new `thumbnail` file field, and cleans up. Frontend reads the thumbnail URL from PocketBase's file API.

**Tech Stack:** Python 3.12 (pypdfium2, Pillow, grpcio-tools), Go 1.25 (PocketBase, gRPC), protobuf, SvelteKit

---

### Task 1: Update proto definition files

**Files:**
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\ocr-server\proto\ocr.proto`
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\internal\ocr\proto\ocr.proto`

- [ ] **Step 1: Add new messages and RPC to both proto files**

Update `ocr-server/proto/ocr.proto`:

```proto
syntax = "proto3";

package ocr;

service OCREngine {
    rpc ExtractText(ExtractRequest) returns (ExtractResponse);
    rpc GenerateThumbnail(ThumbnailRequest) returns (ThumbnailResponse);
}

message ExtractRequest {
    string id = 1;
    string file_path = 2;
    string file_type = 3;
}

message ExtractResponse {
    string text = 1;
}

message ThumbnailRequest {
    string id = 1;
    string file_path = 2;
    string file_type = 3;
}

message ThumbnailResponse {
    string thumbnail_path = 1;
}
```

Update `internal/ocr/proto/ocr.proto` (same content, but keep `option go_package`):

```proto
syntax = "proto3";

package ocr;

option go_package = "SebasXeon/Fileoteca/internal/ocr/proto";

service OCREngine {
    rpc ExtractText(ExtractRequest) returns (ExtractResponse);
    rpc GenerateThumbnail(ThumbnailRequest) returns (ThumbnailResponse);
}

message ExtractRequest {
    string id = 1;
    string file_path = 2;
    string file_type = 3;
}

message ExtractResponse {
    string text = 1;
}

message ThumbnailRequest {
    string id = 1;
    string file_path = 2;
    string file_type = 3;
}

message ThumbnailResponse {
    string thumbnail_path = 1;
}
```

- [ ] **Step 2: Commit**

```bash
git add ocr-server/proto/ocr.proto internal/ocr/proto/ocr.proto
git commit -m "feat(proto): add GenerateThumbnail RPC and ThumbnailRequest/Response messages"
```

---

### Task 2: Regenerate Python proto stubs

**Files:**
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\ocr-server\ocr_server\proto\ocr_pb2.py`
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\ocr-server\ocr_server\proto\ocr_pb2_grpc.py`

- [ ] **Step 1: Regenerate Python protobuf/gRPC code**

```bash
cd ocr-server && uv run python -m grpc_tools.protoc -I proto --python_out=ocr_server/proto --grpc_python_out=ocr_server/proto proto/ocr.proto
```

- [ ] **Step 2: Verify the generated files contain ThumbnailRequest/ThumbnailResponse/GenerateThumbnail**

Check `ocr_server/proto/ocr_pb2_grpc.py` has `GenerateThumbnail` in `OCREngineServicer` and `add_OCREngineServicer_to_server`.

Check `ocr_server/proto/ocr_pb2.py` has `ThumbnailRequest` and `ThumbnailResponse` classes.

- [ ] **Step 3: Fix imports in generated `ocr_pb2_grpc.py`**

The regenerated file may import `ocr_pb2` instead of `from ocr_server.proto import ocr_pb2 as ocr__pb2`. If so, edit the import line in `ocr_server/proto/ocr_pb2_grpc.py`:

```python
from ocr_server.proto import ocr_pb2 as ocr__pb2
```

- [ ] **Step 4: Commit**

```bash
git add ocr-server/ocr_server/proto/ocr_pb2.py ocr-server/ocr_server/proto/ocr_pb2_grpc.py
git commit -m "feat(python): regenerate proto stubs for GenerateThumbnail"
```

---

### Task 3: Create Python thumbnail pipeline module

**Files:**
- Create: `C:\Users\Sebas\Documents\Dev\fileoteca\ocr-server\ocr_server\pipeline\thumbnail.py`

- [ ] **Step 1: Write the thumbnail module**

```python
import tempfile
from pathlib import Path
from PIL import Image

THUMBNAIL_MAX_WIDTH = 400
IMAGE_EXTS = {"png", "jpg", "jpeg", "gif", "bmp", "svg", "webp", "tiff", "ico"}
PDF_EXTS = {"pdf"}


def generate_thumbnail(file_path: str, file_type: str) -> str | None:
    """Generate a thumbnail image (max 400px wide JPEG) from a document.
    Returns temp file path, or None if unsupported."""
    ext = file_type.lower().lstrip(".")

    if ext in PDF_EXTS:
        return _thumbnail_from_pdf(file_path)
    if ext in IMAGE_EXTS:
        return _thumbnail_from_image(file_path)
    return None


def _thumbnail_from_pdf(file_path: str) -> str | None:
    import pypdfium2 as pdfium
    pdf = pdfium.PdfDocument(file_path)
    if len(pdf) == 0:
        return None
    page = pdf[0]
    bitmap = page.render(scale=1)
    pil_image = bitmap.to_pil()
    return _resize_and_save(pil_image)


def _thumbnail_from_image(file_path: str) -> str | None:
    try:
        img = Image.open(file_path)
        if img.mode in ("RGBA", "P", "LA"):
            img = img.convert("RGB")
        return _resize_and_save(img)
    except Exception:
        return None


def _resize_and_save(img: Image.Image) -> str:
    w, h = img.size
    if w > THUMBNAIL_MAX_WIDTH:
        ratio = THUMBNAIL_MAX_WIDTH / w
        new_h = int(h * ratio)
        img = img.resize((THUMBNAIL_MAX_WIDTH, new_h), Image.LANCZOS)
    out_path = str(Path(tempfile.gettempdir()) / f"thumb_{Path(img.filename or 'img').stem}.jpg")
    img.save(out_path, "JPEG", quality=75)
    return out_path
```

- [ ] **Step 2: Commit**

```bash
git add ocr-server/ocr_server/pipeline/thumbnail.py
git commit -m "feat(python): add thumbnail generation pipeline for PDF and images"
```

---

### Task 4: Write Python thumbnail tests

**Files:**
- Create: `C:\Users\Sebas\Documents\Dev\fileoteca\ocr-server\tests\test_thumbnail.py`

- [ ] **Step 1: Write the test file**

```python
import os
import tempfile
from pathlib import Path

import pytest
from PIL import Image

from ocr_server.pipeline.thumbnail import generate_thumbnail


def test_thumbnail_from_image():
    img = Image.new("RGB", (800, 600), color="red")
    tmp = Path(tempfile.gettempdir()) / "test_thumb_src.jpg"
    img.save(tmp, "JPEG")
    try:
        result = generate_thumbnail(str(tmp), "jpg")
        assert result is not None
        assert os.path.exists(result)
        thumb = Image.open(result)
        assert thumb.width <= 400
        assert thumb.height == 300
        os.remove(result)
    finally:
        tmp.unlink(missing_ok=True)


def test_thumbnail_small_image_no_upscale():
    img = Image.new("RGB", (200, 100), color="blue")
    tmp = Path(tempfile.gettempdir()) / "test_thumb_small.png"
    img.save(tmp, "PNG")
    try:
        result = generate_thumbnail(str(tmp), "png")
        assert result is not None
        thumb = Image.open(result)
        assert thumb.width == 200
        os.remove(result)
    finally:
        tmp.unlink(missing_ok=True)


def test_thumbnail_unsupported_returns_none():
    txt = Path(tempfile.gettempdir()) / "test_thumb.txt"
    txt.write_text("hello")
    try:
        result = generate_thumbnail(str(txt), "txt")
        assert result is None
    finally:
        txt.unlink(missing_ok=True)


def test_thumbnail_pdf():
    try:
        import pypdfium2 as pdfium
    except ImportError:
        pytest.skip("pypdfium2 not available")
    pdf = pdfium.PdfDocument.new()
    pdf.new_page(width=595, height=842)
    tmp_pdf = Path(tempfile.gettempdir()) / "test_thumb.pdf"
    pdf.save(str(tmp_pdf))
    try:
        result = generate_thumbnail(str(tmp_pdf), "pdf")
        assert result is not None
        assert os.path.exists(result)
        thumb = Image.open(result)
        assert thumb.width <= 400
        os.remove(result)
    finally:
        tmp_pdf.unlink(missing_ok=True)
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
cd ocr-server && uv run pytest tests/test_thumbnail.py -v
```

Expected: 4 tests pass (or 3 if pypdfium2 test is skipped).

- [ ] **Step 3: Run full test suite to ensure no regressions**

```bash
cd ocr-server && uv run pytest tests/ -v
```

- [ ] **Step 4: Commit**

```bash
git add ocr-server/tests/test_thumbnail.py
git commit -m "test(python): add thumbnail generation tests"
```

---

### Task 5: Add GenerateThumbnail gRPC handler to Python server

**Files:**
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\ocr-server\ocr_server\main.py`

- [ ] **Step 1: Add import for thumbnail module**

In `ocr_server/main.py`, add after the existing pipeline imports (after line 12):

```python
from ocr_server.pipeline import thumbnail as thumbnail_pipeline
```

- [ ] **Step 2: Add GenerateThumbnail method to OCRService class**

In `ocr_server/main.py`, add this method inside the `OCRService` class, after the `ExtractText` method (after line 47):

```python
    def GenerateThumbnail(self, request, context):
        logger.info("Generating thumbnail for %s (%s)", request.id, request.file_type)
        try:
            path = thumbnail_pipeline.generate_thumbnail(request.file_path, request.file_type)
            return ocr_pb2.ThumbnailResponse(thumbnail_path=path or "")
        except Exception as exc:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(exc))
            logger.error("Thumbnail failed for %s: %s", request.id, exc)
            return ocr_pb2.ThumbnailResponse(thumbnail_path="")
```

- [ ] **Step 3: Verify server starts without import errors**

```bash
cd ocr-server && uv run python -m ocr_server.main server --help
```

Expected: help text prints without errors.

- [ ] **Step 4: Commit**

```bash
git add ocr-server/ocr_server/main.py
git commit -m "feat(python): add GenerateThumbnail gRPC handler"
```

---

### Task 6: Regenerate Go proto stubs

**Files:**
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\internal\ocr\proto\ocr.pb.go`
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\internal\ocr\proto\ocr_grpc.pb.go`

- [ ] **Step 1: Regenerate Go protobuf/gRPC code**

```bash
protoc --go_out=. --go-grpc_out=. -I internal/ocr/proto internal/ocr/proto/ocr.proto
```

Note: Run from the project root. Requires `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` installed.

- [ ] **Step 2: Move generated files to correct location if needed**

If protoc outputs to `SebasXeon/Fileoteca/internal/ocr/proto/`, move files:

```bash
# If needed, fix the output path
```

Check: `ocr.pb.go` and `ocr_grpc.pb.go` should be in `internal/ocr/proto/` and contain `ThumbnailRequest`, `ThumbnailResponse`, and `GenerateThumbnail`.

- [ ] **Step 3: Verify Go compiles**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/ocr/proto/ocr.pb.go internal/ocr/proto/ocr_grpc.pb.go
git commit -m "feat(go): regenerate proto stubs for GenerateThumbnail"
```

---

### Task 7: Add thumbnail field to documents collection (migration)

**Files:**
- Create: `C:\Users\Sebas\Documents\Dev\fileoteca\migrations\1776197415_add_thumbnail.go`

- [ ] **Step 1: Write the migration file**

```go
package migrations

import (
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/core"
)

func init() {
	m.Register(func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		// Check if thumbnail field already exists
		for _, f := range documents.Fields {
			if f.GetName() == "thumbnail" {
				return nil
			}
		}

		documents.Fields.Add(&core.FileField{Name: "thumbnail", MaxSelect: 1})
		return app.Save(documents)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		documents.Fields.RemoveByName("thumbnail")
		return app.Save(documents)
	})
}
```

- [ ] **Step 2: Verify Go compiles**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add migrations/1776197415_add_thumbnail.go
git commit -m "feat(db): add thumbnail file field to documents collection"
```

---

### Task 8: Add GenerateThumbnail method to Go OCR client

**Files:**
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\internal\ocr\client.go`

- [ ] **Step 1: Add GenerateThumbnail method to OcrClient**

In `internal/ocr/client.go`, add this method after `ExtractText` (after line 62, before `Close`):

```go
func (c *OcrClient) GenerateThumbnail(ctx context.Context, id, filePath, fileType string) (string, error) {
	req := &proto.ThumbnailRequest{
		Id:       id,
		FilePath: filePath,
		FileType: fileType,
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	resp, err := c.client.GenerateThumbnail(ctx, req)
	if err != nil {
		return "", fmt.Errorf("GenerateThumbnail failed for %s: %w", id, err)
	}

	return resp.ThumbnailPath, nil
}
```

- [ ] **Step 2: Verify Go compiles**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/ocr/client.go
git commit -m "feat(go): add GenerateThumbnail method to OcrClient"
```

---

### Task 9: Integrate thumbnail generation in Go OCR worker

**Files:**
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\internal\ocr\queue.go`

- [ ] **Step 1: Add required imports to queue.go**

Add at top of `import` block in `internal/ocr/queue.go`:

```go
import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)
```

Note: add `"github.com/pocketbase/pocketbase/tools/filesystem"` to existing imports.

- [ ] **Step 2: Modify processJob to generate and upload thumbnail**

In `processJob()`, after the existing OCR success block that calls `w.updateDocumentStatus(...)` and `job.OnComplete(...)`, add thumbnail logic. Replace the entire `processJob` method:

```go
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

	if job.OnComplete != nil {
		job.OnComplete(text)
	}

	w.generateAndUploadThumbnail(job)
}
```

- [ ] **Step 3: Add generateAndUploadThumbnail helper method**

Add to `queue.go` after `updateDocumentStatus`:

```go
func (w *OcrWorker) generateAndUploadThumbnail(job OcrJob) {
	ctx := context.Background()
	thumbPath, err := w.client.GenerateThumbnail(ctx, job.ID, job.FilePath, job.FileType)
	if err != nil {
		log.Printf("Thumbnail generation error for %s: %v", job.ID, err)
		return
	}
	if thumbPath == "" {
		return
	}
	defer os.Remove(thumbPath)

	err = w.app.RunInTransaction(func(txApp core.App) error {
		record, err := txApp.FindRecordById("documents", job.ID)
		if err != nil {
			return fmt.Errorf("record not found %s: %w", job.ID, err)
		}
		file, err := filesystem.NewFileFromPath(thumbPath)
		if err != nil {
			return fmt.Errorf("failed to read thumbnail file: %w", err)
		}
		record.Set("thumbnail", file)
		return txApp.Save(record)
	})
	if err != nil {
		log.Printf("Failed to upload thumbnail for %s: %v", job.ID, err)
	} else {
		log.Printf("Thumbnail saved for document %s", job.ID)
	}
}
```

- [ ] **Step 4: Verify Go compiles**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/ocr/queue.go
git commit -m "feat(go): generate and upload thumbnails after OCR completes"
```

---

### Task 10: Update frontend types and API

**Files:**
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\web\src\lib\types.ts`
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\web\src\lib\api\documents.ts`

- [ ] **Step 1: Add thumbnail to ExplorerFile type**

In `web/src/lib/types.ts`, add `thumbnail?: string` to `ExplorerFile`:

```ts
export type ExplorerFile = {
	id: string;
	name: string;
	ext: FileKind;
	sizeBytes: number;
	updatedAt: Date;
	locationLabel: string;
	category?: string;
	favorite?: boolean;
	thumbnail?: string;
	suggestedReason?: string;
};
```

- [ ] **Step 2: Update toExplorerFile in documents.ts to include thumbnail URL**

In `web/src/lib/api/documents.ts`, modify the `toExplorerFile` function to include thumbnail. Replace the return statement:

```ts
	return {
		id: record.id as string,
		name: record.name as string,
		ext: (record.file_ext ?? "txt") as FileKind,
		sizeBytes: (record.file_size as number) ?? 0,
		updatedAt: new Date(record.updated as string),
		locationLabel: location,
		category: categoryName || undefined,
		favorite: Boolean(record.is_favorite),
		thumbnail: record.thumbnail ? pb.files.getURL(record as any, record.thumbnail as string) : undefined,
	};
```

- [ ] **Step 3: Add thumbnail to DocumentDetail type**

In `web/src/lib/api/documents.ts`, add `thumbnail?: string` to `DocumentDetail`:

```ts
export type DocumentDetail = ExplorerFile & {
	path?: string;
	file?: string;
	notes?: string;
	ocr_txt?: string;
	metadata?: any;
	source_type?: string;
	status?: string;
	last_access?: string;
	category_id?: string;
	subcategory_id?: string;
	thumbnail?: string;
};
```

- [ ] **Step 4: Update getDocument to include thumbnail URL**

In `web/src/lib/api/documents.ts`, in the `getDocument` function return, add:

```ts
		thumbnail: raw.thumbnail ? pb.files.getURL(raw as any, raw.thumbnail as string) : undefined,
```

Place it before the closing `};` in the return object.

- [ ] **Step 5: Verify frontend typechecks**

```bash
cd web && npx svelte-check
```

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/types.ts web/src/lib/api/documents.ts
git commit -m "feat(web): add thumbnail field to ExplorerFile and API"
```

---

### Task 11: Update frontend file-card to show thumbnails

**Files:**
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\web\src\lib\components\explorer\file-card.svelte`

- [ ] **Step 1: Add thumbnail display in file-card**

In `file-card.svelte`, replace the Icon-only divs with conditional thumbnail rendering. The icon area appears in two places: the `suggested` variant and the `else` variant. Replace both icon divs with a component that checks for thumbnail.

In the `suggested` variant (line 60-62), replace:

```svelte
<div class="bg-muted mt-6 flex size-10 items-center justify-center rounded-2xl">
	<Icon class="text-muted-foreground size-5" />
</div>
```

With:

```svelte
<div class="bg-muted mt-6 flex size-10 items-center justify-center rounded-2xl overflow-hidden">
	{#if file.thumbnail}
		<img src={file.thumbnail} alt="" class="size-full object-cover" />
	{:else}
		<Icon class="text-muted-foreground size-5" />
	{/if}
</div>
```

In the default/compact variant (line 82-84), replace:

```svelte
<div class="bg-muted flex size-10 items-center justify-center rounded-2xl" aria-hidden="true">
	<Icon class="text-muted-foreground size-5" />
</div>
```

With:

```svelte
<div class="bg-muted flex size-10 items-center justify-center rounded-2xl overflow-hidden" aria-hidden="true">
	{#if file.thumbnail}
		<img src={file.thumbnail} alt="" class="size-full object-cover" />
	{:else}
		<Icon class="text-muted-foreground size-5" />
	{/if}
</div>
```

- [ ] **Step 2: Verify frontend compiles**

```bash
cd web && npx svelte-check
```

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/components/explorer/file-card.svelte
git commit -m "feat(web): show thumbnail in file-card when available"
```

---

### Task 12: Update frontend document detail page to show thumbnail

**Files:**
- Modify: `C:\Users\Sebas\Documents\Dev\fileoteca\web\src\routes\document\[id]\+page.svelte`

- [ ] **Step 1: Add thumbnail display above document info card**

In `document/[id]/+page.svelte`, inside the info card (`<Card.Root class="shadow-sm h-fit">`), add a thumbnail preview at the top of `Card.Content` (after line 228 `Card.Content` opening). Insert before the first info row:

```svelte
<Card.Content class="flex flex-col gap-3 text-sm">
	{#if doc.thumbnail}
		<div class="rounded-lg overflow-hidden bg-muted">
			<img src={doc.thumbnail} alt={doc.name} class="w-full h-auto object-cover max-h-48" />
		</div>
	{/if}
	<div class="flex justify-between">
```

- [ ] **Step 2: Verify frontend compiles**

```bash
cd web && npx svelte-check
```

- [ ] **Step 3: Commit**

```bash
git add web/src/routes/document/[id]/+page.svelte
git commit -m "feat(web): show thumbnail in document detail view"
```

---

### Task 13: End-to-end verification

- [ ] **Step 1: Start the full application with a test PDF**

```bash
# Build and run from project root
go run . &
```

- [ ] **Step 2: Add a PDF document and verify thumbnail appears**

Add a test PDF. Wait for OCR processing. Check PocketBase admin UI at `http://127.0.0.1:8090/_/` — the document should have a `thumbnail` field with an image.

- [ ] **Step 3: Verify frontend displays thumbnail**

Open the frontend. The file card should show the thumbnail preview instead of the file-type icon. The document detail page should show the thumbnail in the info card.

- [ ] **Step 4: Run full Python test suite**

```bash
cd ocr-server && uv run pytest tests/ -v
```

- [ ] **Step 5: Run Go tests**

```bash
go test ./...
```
