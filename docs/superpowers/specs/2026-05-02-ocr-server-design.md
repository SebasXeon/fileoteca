# OCR Server Design

**Date:** 2026-05-02
**Status:** Approved
**Context:** [2026-04-30-fileoteca-windows-integration-design.md](./2026-04-30-fileoteca-windows-integration-design.md)

## Overview

Build a Python gRPC OCR microservice that extracts text from documents (images, PDFs, Office files). The Go backend starts the Python server as a subprocess and communicates via gRPC. OCR runs asynchronously in background — the document appears instantly in the UI and `ocr_txt` is filled later.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│ Fileoteca.exe (Go)                                  │
│                                                     │
│  docadd ──▶ ocrQueue (channel) ──▶ OcrWorker ──▶ PD│
│            (status: "pending")   (goroutine)   updat│
│                                                     │
│  exec.Command("uv", "run", "ocr-server")            │
└──────────────────┬──────────────────────────────────┘
                   │ gRPC (localhost:50051)
┌──────────────────▼──────────────────────────────────┐
│ ocr-server (Python, UV)                             │
│                                                     │
│  gRPC Server → Semaphore(1) → Pipeline → OCREngine │
│                                                     │
│  Pipeline:                                          │
│    PDF   → pypdfium2 → images → WinOCR             │
│    Image → WinOCR                                   │
│    Doc   → MarkItDown → strip markdown → text       │
│                                                     │
│  PDF page limit: first 5 + last 2 (if pages > 7)   │
└─────────────────────────────────────────────────────┘
```

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **gRPC contract** | Single RPC `ExtractText(file_path) -> text` | Simple, no metadata needed yet |
| **Process lifecycle** | Go starts Python as subprocess via UV | Go is the process manager; single-binary UX |
| **Concurrency** | Dual protection: Go internal channel + Python `max_workers=1` | Safety on both sides; 1 doc at a time guaranteed |
| **File resolution** | Go resolves path (local → direct; PB storage → copy to %TEMP%) | OCR server only deals with readable paths |
| **OCR trigger** | Async: doc created immediately, OCR worker runs in background | Best UX; doc appears instantly |
| **Engine selection** | Config file (`config.toml`) | Changed via config + restart; no per-request override |
| **CLI mode** | Subcommand: `uv run ocr-server` vs `uv run ocr-server ocr <file>` | Clean separation of server/one-shot modes |
| **Dev packaging** | Go uses `uv run --directory <path>/ocr-server` relative to working dir | Simple for development |
| **Page extraction** | First 5 + last 2 pages for PDFs with >7 pages | Balances speed vs completeness |
| **Engine architecture** | Registry pattern with decorator `@register("name")` | Add new engines without modifying existing code |

## gRPC Contract

```protobuf
syntax = "proto3";

service OCREngine {
    rpc ExtractText(ExtractRequest) returns (ExtractResponse);
}

message ExtractRequest {
    string id = 1;          // document ID for tracing
    string file_path = 2;   // absolute path to readable temp file
    string file_type = 3;   // "pdf", "docx", "png", "jpg", etc.
}

message ExtractResponse {
    string text = 1;        // extracted plain text
}
```

## Python Project Structure

```
ocr-server/
├── ocr_server/
│   ├── __init__.py
│   ├── main.py                       # Entrypoint: gRPC server / CLI
│   ├── proto/
│   │   ├── ocr.proto                 # gRPC service definition
│   │   ├── ocr_pb2.py               # Generated
│   │   └── ocr_pb2_grpc.py          # Generated
│   ├── engine/
│   │   ├── __init__.py
│   │   ├── base.py                   # ABC OCREngine
│   │   ├── registry.py              # Engine registry + factory
│   │   └── winocr.py                # WinOCREngine (default, only impl for now)
│   ├── pipeline/
│   │   ├── __init__.py
│   │   ├── pdf.py                   # PDF → images via pypdfium2 (page selection)
│   │   ├── markitdown_.py           # Office docs → text via MarkItDown
│   │   └── image.py                 # Image → text via registered engine
│   └── queue.py                     # ThreadPoolExecutor(max_workers=1)
├── config.toml                      # engine = "winocr", port = 50051
├── tests/
├── pyproject.toml
├── uv.lock
├── .python-version
```

## Engine Pluggability

```python
# engine/registry.py
_engines: dict[str, type[OCREngine]] = {}

def register(name: str):
    def decorator(cls):
        _engines[name] = cls
        return cls
    return decorator

def create(name: str) -> OCREngine:
    return _engines[name]()
```

```python
# engine/base.py
class OCREngine(ABC):
    @abstractmethod
    def extract_from_image(self, image_path: str) -> str: ...

    def extract_from_images(self, image_paths: list[str]) -> str:
        texts = [self.extract_from_image(p) for p in image_paths]
        return "\n---\n".join(texts)
```

Adding a new engine (e.g., Tesseract):

```python
@register("tesseract")
class TesseractEngine(OCREngine):
    def extract_from_image(self, image_path: str) -> str: ...
```

Config selects: `engine = "tesseract"` in `config.toml`.

## Pipeline Logic

```
ExtractText(file_path, file_type)
  ├── file_type is pdf       → pdf_to_images(path) → select pages → engine.extract_from_images()
  ├── file_type is image     → engine.extract_from_image(path)
  └── file_type is doc/other → markitdown convert(path) → strip markdown → plain text
```

PDF page selection:
```python
MAX_HEAD = 5   # first N pages
MAX_TAIL = 2   # last N pages

def select_pages(total: int) -> list[int]:
    if total <= MAX_HEAD + MAX_TAIL:
        return list(range(1, total + 1))
    return list(range(1, MAX_HEAD + 1)) + list(range(total - MAX_TAIL + 1, total + 1))
```

## Go Integration

### New files

| File | Purpose |
|------|---------|
| `internal/ocr/client.go` | gRPC client wrapper (`OcrClient`): connect, `ExtractText`, close |
| `internal/ocr/queue.go` | `OcrWorker`: goroutine with channel, file path resolution, temp file management, DB update |
| `internal/ocr/server.go` | Python subprocess lifecycle: start/stop via `exec.Command("uv", ...)` |

### Flow

1. `main.go` starts `OcrServer` (Python subprocess via `exec.Command`)
2. Creates `OcrClient` (gRPC dial `localhost:50051` with retry)
3. Starts `OcrWorker` goroutine consuming from channel
4. `docadd.go` enqueues job: `ocrQueue <- OcrJob{ID, ResolvedPath, FileType}` — non-blocking
5. Worker picks job, resolves path (local or PB storage→temp copy), calls `ExtractText` gRPC
6. On success: updates doc `ocr_txt` + `status: "processed"`
7. On error: updates `status: "error"` + logs error
8. Cleans up temp file if one was created
9. On shutdown: closes channel, closes gRPC conn, kills Python subprocess

### DB fields updated

| Field | Set to |
|-------|--------|
| `ocr_txt` | Extracted text string |
| `status` | `"processed"` or `"error"` |

### Path resolution

```
if doc.path != "" && os.FileExists(doc.path):
    return doc.path                    // direct local file
elif doc.file != "":
    // file is in pb_data/storage/<collection>/<id>/<filename>
    // copy to os.TempDir()/fileoteca/<id>.<ext>
    return tempPath
```

## CLI Mode

```
# Start gRPC server (default)
uv run ocr-server

# One-shot OCR on a file (for testing)
uv run ocr-server ocr path/to/doc.pdf
uv run ocr-server ocr path/to/image.png
uv run ocr-server ocr path/to/file.docx
```

CLI output: plain text to stdout, errors to stderr.

## Dependencies

### Python (added vs current pyproject.toml)

| Package | Version | Purpose |
|---------|---------|---------|
| `grpcio` | >=1.71 | gRPC runtime |
| `grpcio-tools` | >=1.71 | Proto compiler |
| `markitdown` | (latest) | Office document → text conversion |
| (existing) `pillow` | >=12.2.0 | Image handling |
| (existing) `pypdfium2` | >=5.7.1 | PDF → image rendering |
| (existing) `winrt-windows-media-ocr` | >=3.2.1 | Windows OCR engine |
| (existing) `winrt-windows-graphics-imaging` | >=3.2.1 | Image format conversion |
| (existing) `winrt-windows-storage-streams` | >=3.2.1 | Stream handling |

### Go (added)

| Package | Purpose |
|---------|---------|
| `google.golang.org/grpc` | gRPC client |
| `google.golang.org/protobuf` | Proto message types |

## Error Handling

- **gRPC connection failure**: Retry with exponential backoff on startup; mark job as error if unreachable during processing
- **OCR engine failure**: Catch exception, return empty text, Go sets `status: "error"`
- **PDF rendering failure**: Per-page try/catch, skip failing pages, continue with others
- **MarkItDown failure**: Return error, Go sets `status: "error"`
- **Temp file cleanup**: Always cleanup in `defer`, even on error paths

## Testing

- **CLI mode**: Manual testing with real documents via `uv run ocr-server ocr <file>`
- **Python unit tests**: `tests/test_pipeline.py`, `tests/test_engine_winocr.py` with sample images
- **Go integration**: Can be tested manually by adding documents through the UI/context menu
- **gRPC health check**: Simple `Health` RPC can be added later for readiness probes

## Future Extensions (out of scope)

- Per-request engine override
- Page range configuration (beyond 5+2)
- Confidence scores and bounding boxes
- Health/readiness gRPC endpoint
- `Tesseract` engine (via pytesseract)
- `SuryaOCR` / `PaddleOCR` engines
- Empaquetado: embed `ocr-server/` in Go binary with `embed.FS`
