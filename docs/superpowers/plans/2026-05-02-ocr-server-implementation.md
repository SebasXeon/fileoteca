# OCR Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Python gRPC OCR microservice with pluggable engines, and integrate it into the Go backend so documents get OCR-processed asynchronously on creation.

**Architecture:** Python gRPC server (`ocr-server/`) using WinOCR + pypdfium2 + MarkItDown. Go backend launches it as subprocess via `exec.Command`, registers a PocketBase `OnRecordCreate` hook that enqueues OCR jobs. A worker goroutine with a buffered channel serializes processing, resolves file paths (local or PB storage→temp), calls gRPC, and updates `ocr_txt` + `status` on the document record.

**Tech Stack:** Python 3.14 (UV), gRPC, winrt, pypdfium2, MarkItDown, Go 1.25, google.golang.org/grpc, google.golang.org/protobuf

---

### Task 1: Add gRPC and MarkItDown dependencies

**Files:**
- Modify: `ocr-server/pyproject.toml`

- [ ] **Step 1: Update pyproject.toml**

Add new dependencies to the existing `dependencies` list in `ocr-server/pyproject.toml`:

```toml
[project]
name = "ocr-server"
version = "0.1.0"
requires-python = ">=3.14"
dependencies = [
    "pillow>=12.2.0",
    "pypdfium2>=5.7.1",
    "winrt-runtime>=3.2.1",
    "winrt-windows-globalization>=3.2.1",
    "winrt-windows-graphics-imaging>=3.2.1",
    "winrt-windows-media-ocr>=3.2.1",
    "winrt-windows-storage>=3.2.1",
    "winrt-windows-storage-streams>=3.2.1",
    "grpcio>=1.71.0",
    "grpcio-tools>=1.71.0",
    "markitdown>=0.0.1",
    "tomli>=2.0.0",
]
```

- [ ] **Step 2: Lock dependencies with UV**

```powershell
uv lock
```
Workdir: `ocr-server/`

Expected: `uv.lock` updated without errors.

- [ ] **Step 3: Commit**

```bash
git add ocr-server/pyproject.toml ocr-server/uv.lock
git commit -m "deps: add grpcio, grpcio-tools, markitdown, tomli to ocr-server"
```

---

### Task 2: Create proto file and generate Python stubs

**Files:**
- Create: `ocr-server/proto/ocr.proto`
- Create: `ocr-server/proto/__init__.py` (empty)
- Generate: `ocr-server/ocr_server/proto/ocr_pb2.py`
- Generate: `ocr-server/ocr_server/proto/ocr_pb2_grpc.py`

- [ ] **Step 1: Create proto directory and proto file**

```bash
mkdir ocr-server/proto
```

```protobuf
syntax = "proto3";

package ocr;

service OCREngine {
    rpc ExtractText(ExtractRequest) returns (ExtractResponse);
}

message ExtractRequest {
    string id = 1;
    string file_path = 2;
    string file_type = 3;
}

message ExtractResponse {
    string text = 1;
}
```

Write to `ocr-server/proto/ocr.proto`.

- [ ] **Step 2: Create empty __init__.py**

```bash
mkdir ocr-server/ocr_server/proto 2>$null; echo "" > ocr-server/ocr_server/proto/__init__.py
echo "" > ocr-server/proto/__init__.py
```

Workdir: `ocr-server/`

- [ ] **Step 3: Generate Python proto stubs**

```powershell
uv run python -m grpc_tools.protoc -Iproto --python_out=ocr_server/proto --grpc_python_out=ocr_server/proto proto/ocr.proto
```
Workdir: `ocr-server/`

Expected: Creates `ocr_pb2.py` and `ocr_pb2_grpc.py` in `ocr-server/ocr_server/proto/`.

- [ ] **Step 4: Commit**

```bash
git add ocr-server/proto/ ocr-server/ocr_server/proto/
git commit -m "feat(ocr): add proto definition and generate Python stubs"
```

---

### Task 3: Engine base class and registry

**Files:**
- Create: `ocr-server/ocr_server/engine/__init__.py`
- Create: `ocr-server/ocr_server/engine/base.py`
- Create: `ocr-server/ocr_server/engine/registry.py`

- [ ] **Step 1: Create engine package init**

```python
# ocr-server/ocr_server/engine/__init__.py
from .registry import create_engine
from .base import OCREngine

__all__ = ["OCREngine", "create_engine"]
```

- [ ] **Step 2: Create base engine ABC**

```python
# ocr-server/ocr_server/engine/base.py
from abc import ABC, abstractmethod


class OCREngine(ABC):
    """Abstract base for OCR engines."""

    @abstractmethod
    def extract_from_image(self, image_path: str) -> str:
        """Extract text from a single image file path.

        Returns the recognized text.
        """
        ...

    def extract_from_images(self, image_paths: list[str]) -> str:
        """Extract text from multiple images, concatenating results."""
        texts: list[str] = []
        for i, path in enumerate(image_paths, 1):
            t = self.extract_from_image(path)
            if t.strip():
                texts.append(f"[Página {i}]\n{t}")
        return "\n---\n".join(texts) if texts else ""
```

- [ ] **Step 3: Create engine registry**

```python
# ocr-server/ocr_server/engine/registry.py
from .base import OCREngine

_engines: dict[str, type[OCREngine]] = {}


def register(name: str):
    """Decorator that registers an OCREngine subclass under the given name."""
    def decorator(cls: type[OCREngine]) -> type[OCREngine]:
        _engines[name] = cls
        return cls
    return decorator


def create_engine(name: str) -> OCREngine:
    """Factory: instantiate a registered engine by name."""
    if name not in _engines:
        raise ValueError(f"Unknown OCR engine: {name!r}. Available: {list(_engines.keys())}")
    return _engines[name]()


def available_engines() -> list[str]:
    """Return list of registered engine names."""
    return list(_engines.keys())
```

- [ ] **Step 4: Commit**

```bash
git add ocr-server/ocr_server/engine/
git commit -m "feat(ocr): add OCREngine base class and registry pattern"
```

---

### Task 4: WinOCREngine

**Files:**
- Create: `ocr-server/ocr_server/engine/winocr.py`

- [ ] **Step 1: Implement WinOCREngine**

```python
# ocr-server/ocr_server/engine/winocr.py
import asyncio
from PIL import Image
import winrt.windows.storage.streams as streams
from winrt.windows.media.ocr import OcrEngine
from winrt.windows.graphics.imaging import SoftwareBitmap, BitmapPixelFormat

from .registry import register


def _pil_to_software_bitmap(path: str) -> SoftwareBitmap:
    img = Image.open(path).convert("RGBA")
    writer = streams.DataWriter()
    writer.write_bytes(img.tobytes())
    bitmap = SoftwareBitmap(BitmapPixelFormat.RGBA8, img.width, img.height)
    bitmap.copy_from_buffer(writer.detach_buffer())
    return bitmap


from .base import OCREngine


@register("winocr")
class WinOCREngine(OCREngine):
    def extract_from_image(self, image_path: str) -> str:
        bitmap = _pil_to_software_bitmap(image_path)
        engine = OcrEngine.try_create_from_user_profile_languages()
        if engine is None:
            raise RuntimeError("No OCR engine available for this language")
        result = asyncio.run(engine.recognize_async(bitmap))
        return result.text if result else ""
```

- [ ] **Step 2: Commit**

```bash
git add ocr-server/ocr_server/engine/winocr.py
git commit -m "feat(ocr): implement WinOCREngine"
```

---

### Task 5: PDF pipeline (page selection + render)

**Files:**
- Create: `ocr-server/ocr_server/pipeline/__init__.py`
- Create: `ocr-server/ocr_server/pipeline/pdf.py`

- [ ] **Step 1: Create pipeline package init**

```python
# ocr-server/ocr_server/pipeline/__init__.py
```

- [ ] **Step 2: Implement PDF → image rendering**

```python
# ocr-server/ocr_server/pipeline/pdf.py
import tempfile
from pathlib import Path
import pypdfium2 as pdfium

HEAD_PAGES = 5
TAIL_PAGES = 2


def pdf_to_images(file_path: str, temp_dir: str | None = None) -> list[str]:
    """Render selected pages of a PDF to temporary PNG images.

    Returns a list of file paths to the generated images.
    If the PDF has <= 7 pages, all pages are rendered.
    If > 7, only the first 5 and last 2 pages are rendered.
    """
    parent = temp_dir or tempfile.gettempdir()
    pdf = pdfium.PdfDocument(file_path)
    total = len(pdf)

    pages_to_render = _select_pages(total)

    image_paths: list[str] = []
    for page_num in pages_to_render:
        page = pdf[page_num - 1]
        bitmap = page.render(scale=2)
        pil_image = bitmap.to_pil()
        out_path = str(Path(parent) / f"ocr_page_{page_num:04d}.png")
        pil_image.save(out_path, "PNG")
        image_paths.append(out_path)

    return image_paths


def _select_pages(total: int) -> list[int]:
    if total <= HEAD_PAGES + TAIL_PAGES:
        return list(range(1, total + 1))
    return list(range(1, HEAD_PAGES + 1)) + list(range(total - TAIL_PAGES + 1, total + 1))
```

- [ ] **Step 3: Commit**

```bash
git add ocr-server/ocr_server/pipeline/
git commit -m "feat(ocr): implement PDF to images pipeline with page selection"
```

---

### Task 6: MarkItDown pipeline for Office docs

**Files:**
- Create: `ocr-server/ocr_server/pipeline/markitdown_.py`

- [ ] **Step 1: Implement MarkItDown conversion**

```python
# ocr-server/ocr_server/pipeline/markitdown_.py
from markitdown import MarkItDown


def extract_text(file_path: str) -> str:
    """Extract plain text from an Office document using MarkItDown.

    Supports: .docx, .pptx, .xlsx, .doc, .ppt, .xls, .odt, .ods, .odp,
    .html, .htm, .csv, .json, .xml, .md, .rtf, .txt
    """
    md = MarkItDown()
    result = md.convert(file_path)
    return result.text_content if result else ""
```

- [ ] **Step 2: Commit**

```bash
git add ocr-server/ocr_server/pipeline/markitdown_.py
git commit -m "feat(ocr): add MarkItDown pipeline for Office docs"
```

---

### Task 7: Image OCR pipeline

**Files:**
- Create: `ocr-server/ocr_server/pipeline/image.py`

- [ ] **Step 1: Implement image OCR pipeline**

```python
# ocr-server/ocr_server/pipeline/image.py
import tempfile
from pathlib import Path

from ..engine import create_engine


def extract_text(image_paths: list[str], engine_name: str, temp_dir: str | None = None) -> str:
    """Extract text from a list of image paths using the specified OCR engine.

    For multiple images, text is concatenated with page separators.
    """
    engine = create_engine(engine_name)
    return engine.extract_from_images(image_paths)
```

- [ ] **Step 2: Commit**

```bash
git add ocr-server/ocr_server/pipeline/image.py
git commit -m "feat(ocr): add image OCR pipeline"
```

---

### Task 8: Queue (concurrency control)

**Files:**
- Create: `ocr-server/ocr_server/queue.py`

- [ ] **Step 1: Implement sequential processing queue**

```python
# ocr-server/ocr_server/queue.py
from concurrent.futures import ThreadPoolExecutor, Future


class OCRQueue:
    """Ensures only one OCR job runs at a time."""

    def __init__(self) -> None:
        self._executor = ThreadPoolExecutor(max_workers=1)

    def submit(self, fn, *args, **kwargs) -> Future:
        return self._executor.submit(fn, *args, **kwargs)

    def shutdown(self) -> None:
        self._executor.shutdown(wait=True)
```

- [ ] **Step 2: Commit**

```bash
git add ocr-server/ocr_server/queue.py
git commit -m "feat(ocr): add sequential processing queue"
```

---

### Task 9: gRPC server + config loading + request handler

**Files:**
- Modify: `ocr-server/ocr_server/__init__.py`
- Create: `ocr-server/config.toml`
- Modify: `ocr-server/ocr_server/main.py` (complete rewrite of stub)

- [ ] **Step 1: Create ocr_server package init** (or verify it exists)

```python
# ocr-server/ocr_server/__init__.py
```

- [ ] **Step 2: Create config.toml**

```toml
# ocr-server/config.toml
engine = "winocr"
host = "[::1]"
port = 50051
```

- [ ] **Step 3: Implement main.py server + handler**

```python
# ocr-server/ocr_server/main.py
import argparse
import logging
import sys
from concurrent import futures
from pathlib import Path

import grpc
import tomli

from ocr_server.proto import ocr_pb2, ocr_pb2_grpc
from ocr_server.pipeline import pdf as pdf_pipeline
from ocr_server.pipeline import image as image_pipeline
from ocr_server.pipeline import markitdown_
from ocr_server.queue import OCRQueue

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("ocr-server")

IMAGE_EXTS = {"png", "jpg", "jpeg", "gif", "bmp", "svg", "webp", "tiff", "ico"}
PDF_EXTS = {"pdf"}


def load_config() -> dict:
    config_path = Path(__file__).parent.parent / "config.toml"
    if not config_path.exists():
        return {"engine": "winocr", "host": "[::1]", "port": 50051}
    with open(config_path, "rb") as f:
        return tomli.load(f)


class OCRService(ocr_pb2_grpc.OCREngineServicer):
    def __init__(self, engine_name: str, queue: OCRQueue) -> None:
        self._engine = engine_name
        self._queue = queue

    def ExtractText(self, request, context):
        logger.info("Processing document %s (%s)", request.id, request.file_type)
        try:
            future = self._queue.submit(
                _extract_text, request.id, request.file_path, request.file_type, self._engine
            )
            text = future.result(timeout=300)
            return ocr_pb2.ExtractResponse(text=text)
        except Exception as exc:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(exc))
            logger.error("OCR failed for %s: %s", request.id, exc)
            return ocr_pb2.ExtractResponse(text="")


def _extract_text(doc_id: str, file_path: str, file_type: str, engine: str) -> str:
    ext = file_type.lower().lstrip(".")

    if ext in PDF_EXTS:
        logger.info("Rendering PDF pages for %s", doc_id)
        image_paths = pdf_pipeline.pdf_to_images(file_path)
        if not image_paths:
            return ""
        result = image_pipeline.extract_text(image_paths, engine)
        # Cleanup temp images
        for p in image_paths:
            Path(p).unlink(missing_ok=True)
        return result

    if ext in IMAGE_EXTS:
        return image_pipeline.extract_text([file_path], engine)

    # Office / text / unknown → MarkItDown
    return markitdown_.extract_text(file_path)


def run_server(config: dict | None = None) -> None:
    if config is None:
        config = load_config()

    host = config.get("host", "[::1]")
    port = config.get("port", 50051)
    engine_name = config.get("engine", "winocr")
    address = f"{host}:{port}"

    queue = OCRQueue()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    ocr_pb2_grpc.add_OCREngineServicer_to_server(OCRService(engine_name, queue), server)
    server.add_insecure_port(address)
    server.start()
    logger.info("OCR server started on %s (engine: %s)", address, engine_name)
    server.wait_for_termination()
    queue.shutdown()


def run_cli(file_path: str, config: dict | None = None) -> None:
    if config is None:
        config = load_config()
    engine_name = config.get("engine", "winocr")
    file_type = Path(file_path).suffix
    text = _extract_text("cli", file_path, file_type, engine_name)
    if text.strip():
        print(text)
    else:
        print("(no text extracted)", file=sys.stderr)
        sys.exit(1)


def main() -> None:
    parser = argparse.ArgumentParser(description="OCR gRPC server")
    sub = parser.add_subparsers(dest="command")

    server_parser = sub.add_parser("server", help="Start gRPC server")
    server_parser.add_argument("--config", type=str, default=None, help="Path to config.toml")

    ocr_parser = sub.add_parser("ocr", help="One-shot OCR on a file")
    ocr_parser.add_argument("file", type=str, help="Path to file")
    ocr_parser.add_argument("--config", type=str, default=None, help="Path to config.toml")

    args = parser.parse_args()

    config = {}
    if hasattr(args, "config") and args.config:
        with open(args.config, "rb") as f:
            config = tomli.load(f)

    if args.command == "ocr":
        run_cli(args.file, config)
    else:
        # Default: server mode
        run_server(config)


if __name__ == "__main__":
    main()
```

- [ ] **Step 4: Validate the module loads**

```powershell
uv run python -c "from ocr_server.main import main; print('import OK')"
```
Workdir: `ocr-server/`

- [ ] **Step 5: Commit**

```bash
git add ocr-server/ocr_server/ ocr-server/config.toml
git commit -m "feat(ocr): implement gRPC server with config and request handler"
```

---

### Task 10: Update entrypoint script

**Files:**
- Modify: `ocr-server/main.py` (update stub at repo root)
- Create/verify: `ocr-server/pyproject.toml` `[project.scripts]`
- Rename: Remove old `ocr-server/winocr.py` and `ocr-server/pdfexample.py`

- [ ] **Step 1: Update pyproject.toml with script entrypoint**

Add to `ocr-server/pyproject.toml`:

```toml
[project.scripts]
ocr-server = "ocr_server.main:main"
```

- [ ] **Step 2: Update main.py at repo root to be a redirect**

```python
# ocr-server/main.py
from ocr_server.main import main

if __name__ == "__main__":
    main()
```

- [ ] **Step 3: Remove old example files**

```bash
Remove-Item ocr-server/winocr.py
Remove-Item ocr-server/pdfexample.py
```
Workdir: repo root

- [ ] **Step 4: Test CLI mode**

```powershell
uv run ocr-server ocr samples/page_00000001.jpg
```
Workdir: `ocr-server/`

Expected: Outputs recognized text from the sample image.

- [ ] **Step 5: Test server mode starts and listens**

```powershell
uv run python -c "import grpc; ch = grpc.insecure_channel('localhost:50051')"
```
Before running: start `uv run ocr-server` in another terminal, wait 3 seconds, then run the test.

- [ ] **Step 6: Commit**

```bash
git add ocr-server/main.py ocr-server/pyproject.toml
git rm ocr-server/winocr.py ocr-server/pdfexample.py
git commit -m "feat(ocr): wire up entrypoint, remove old examples"
```

---

### Task 11: Python unit tests

**Files:**
- Create: `ocr-server/tests/__init__.py`
- Create: `ocr-server/tests/test_pipeline.py`
- Create: `ocr-server/tests/test_engine.py`

- [ ] **Step 1: Create tests init**

```python
# ocr-server/tests/__init__.py
```

- [ ] **Step 2: Test PDF page selection logic**

```python
# ocr-server/tests/test_pipeline.py
import pytest
from ocr_server.pipeline.pdf import _select_pages


def test_select_pages_small_pdf():
    assert _select_pages(3) == [1, 2, 3]
    assert _select_pages(7) == [1, 2, 3, 4, 5, 6, 7]
    assert _select_pages(1) == [1]


def test_select_pages_large_pdf():
    assert _select_pages(10) == [1, 2, 3, 4, 5, 9, 10]
    assert _select_pages(20) == [1, 2, 3, 4, 5, 19, 20]
    assert _select_pages(100) == [1, 2, 3, 4, 5, 99, 100]
```

- [ ] **Step 3: Test engine registry**

```python
# ocr-server/tests/test_engine.py
import pytest
from ocr_server.engine.registry import create_engine, available_engines


def test_registry_has_winocr():
    import ocr_server.engine.winocr  # register
    engs = available_engines()
    assert "winocr" in engs


def test_unknown_engine_raises():
    with pytest.raises(ValueError, match="Unknown OCR engine"):
        create_engine("nonexistent")
```

- [ ] **Step 4: Run tests**

```powershell
uv run pytest tests/ -v
```
Workdir: `ocr-server/`

Expected: 4 tests pass.

- [ ] **Step 5: Commit**

```bash
git add ocr-server/tests/
git commit -m "test(ocr): add pipeline and engine unit tests"
```

---

### Task 12: Add gRPC Go dependency and proto types

**Files:**
- Modify: `go.mod`
- Create: `internal/ocr/proto/ocr.proto`
- Generate: `internal/ocr/proto/ocr.pb.go`, `internal/ocr/proto/ocr_grpc.pb.go`

- [ ] **Step 1: Add gRPC dependency to go.mod**

```powershell
go get google.golang.org/grpc@latest
go get google.golang.org/protobuf@latest
go mod tidy
```
Workdir: repo root

- [ ] **Step 2: Install protoc Go plugins**

```powershell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Ensure `$env:GOPATH\bin` (usually `%USERPROFILE%\go\bin`) is in PATH.

- [ ] **Step 3: Create proto directory for Go**

```bash
mkdir internal\ocr\proto 2>$null
```

Workdir: repo root.

Copy the proto file:
```bash
copy ocr-server\proto\ocr.proto internal\ocr\proto\ocr.proto
```

Update the go_package option in the Go copy:

```protobuf
syntax = "proto3";

package ocr;

option go_package = "SebasXeon/Fileoteca/internal/ocr/proto";

service OCREngine {
    rpc ExtractText(ExtractRequest) returns (ExtractResponse);
}

message ExtractRequest {
    string id = 1;
    string file_path = 2;
    string file_type = 3;
}

message ExtractResponse {
    string text = 1;
}
```

Write this to `internal/ocr/proto/ocr.proto`.

- [ ] **Step 4: Generate Go proto code**

First, install protoc itself if not present:

```powershell
# Option A: via winget
winget install --id Google.Protobuf 2>$null

# Option B: manual download
$url = "https://github.com/protocolbuffers/protobuf/releases/download/v30.2/protoc-30.2-win64.zip"
Invoke-WebRequest -Uri $url -OutFile "$env:TEMP\protoc.zip"
Expand-Archive "$env:TEMP\protoc.zip" -DestinationPath "$env:TEMP\protoc"
$env:PATH = "$env:TEMP\protoc\bin;$env:PATH"
```

Then generate:
```powershell
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/ocr/proto/ocr.proto
```
Workdir: repo root.

Expected: Creates `internal/ocr/proto/ocr.pb.go` and `internal/ocr/proto/ocr_grpc.pb.go`.

- [ ] **Step 5: Verify Go compiles**

```powershell
go build ./...
```
Workdir: repo root.

- [ ] **Step 6: Commit**

```bash
git add internal/ocr/ go.mod go.sum
git commit -m "deps: add grpc Go dependencies and generate proto stubs"
```

---

### Task 13: gRPC client wrapper

**Files:**
- Create: `internal/ocr/client.go`

- [ ] **Step 1: Implement OcrClient**

```go
// internal/ocr/client.go
package ocr

import (
	"context"
	"fmt"
	"log"
	"time"

	"SebasXeon/Fileoteca/internal/ocr/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OcrClient struct {
	conn   *grpc.ClientConn
	client proto.OCREngineClient
	addr   string
}

func NewOcrClient(addr string) (*OcrClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OCR server at %s: %w", addr, err)
	}

	client := proto.NewOCREngineClient(conn)
	log.Printf("connected to OCR server at %s", addr)

	return &OcrClient{
		conn:   conn,
		client: client,
		addr:   addr,
	}, nil
}

func (c *OcrClient) ExtractText(ctx context.Context, id, filePath, fileType string) (string, error) {
	req := &proto.ExtractRequest{
		Id:       id,
		FilePath: filePath,
		FileType: fileType,
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	resp, err := c.client.ExtractText(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OCR ExtractText failed for %s: %w", id, err)
	}

	return resp.Text, nil
}

func (c *OcrClient) Close() error {
	return c.conn.Close()
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/ocr/client.go
git commit -m "feat(ocr): add gRPC client wrapper"
```

---

### Task 14: OCR worker (queue, path resolution, DB update)

**Files:**
- Create: `internal/ocr/queue.go`

- [ ] **Step 1: Implement OcrWorker**

```go
// internal/ocr/queue.go
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

	pbocr "SebasXeon/Fileoteca/internal/ocr/proto"
)

type OcrJob struct {
	ID       string
	FilePath string // resolved readable path
	FileType string
}

type OcrWorker struct {
	client  *OcrClient
	jobs    chan OcrJob
	app     *pocketbase.PocketBase
	quit    chan struct{}
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
// For local files (path field is set), returns it directly.
// For uploaded files (file field in PB storage), copies to temp dir.
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

	// Build PB storage path: pb_data/storage/<collection_id>/<record_id>/<filename>
	col, err := record.Collection()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get collection for %s: %w", record.Id, err)
	}

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
```

- [ ] **Step 2: Commit**

```bash
git add internal/ocr/queue.go
git commit -m "feat(ocr): add OCR worker with queue, path resolution, DB update"
```

---

### Task 15: Python subprocess lifecycle

**Files:**
- Create: `internal/ocr/server.go`

- [ ] **Step 1: Implement Python server lifecycle manager**

```go
// internal/ocr/server.go
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

// StartOcrServer launches the Python OCR server via UV.
// ocrServerDir is the path to the ocr-server/ directory.
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

	// Wait for server to be ready
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
```

- [ ] **Step 2: Commit**

```bash
git add internal/ocr/server.go
git commit -m "feat(ocr): add Python subprocess lifecycle manager"
```

---

### Task 16: Integrate into main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Modify main.go to integrate OCR**

Replace `main.go` with this full version:

```go
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

			app.OnRecordCreate("documents").BindFunc(func(e *core.RecordCreateEvent) error {
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
```

---

- [ ] **Step 2: Verify Go compiles**

```powershell
go build -o Fileoteca.exe .
```
Workdir: repo root.

Expected: Build succeeds. 0 errors.

- [ ] **Step 3: Commit**

```bash
git add main.go internal/shell/server.go
git commit -m "feat(ocr): integrate OCR worker and record-create hook into main"
```

---

### Task 17: End-to-end manual test

- [ ] **Step 1: Start the full stack**

Terminal 1 — OCR server:
```powershell
cd ocr-server
uv run ocr-server
```
Expected: `OCR server started on [::1]:50051 (engine: winocr)`

Terminal 2 — Go backend:
```powershell
go run .
```
Expected: `OCR server started (PID ...)`, `connected to OCR server at localhost:50051`, `OCR integrated successfully`, `Fileoteca iniciada...`

- [ ] **Step 2: Test with CLI mode**

```powershell
cd ocr-server
uv run ocr-server ocr samples/page_00000001.jpg
```
Expected: Recognized text printed to stdout.

- [ ] **Step 3: Test via context menu (if available)**

Right-click a PDF/image in Explorer → "Agregar a Fileoteca"
Expected: Document appears in UI with status "pending", then after a few seconds, `ocr_txt` is populated and status becomes "processed".

- [ ] **Step 4: Check PocketBase admin**

Open http://127.0.0.1:8090/_/ → documents collection → verify `ocr_txt` and `status` fields are updated.

- [ ] **Step 5: Commit any final fixes**

```bash
git add -A
git commit -m "chore(ocr): final integration fixes and cleanup"
```

---

### Task 18: Remove old `ocr-server/main.py` duplicate and gitignore venv/build artifacts

**Files:**
- Verify: `ocr-server/main.py` only has the redirect code
- Create/Modify: `.gitignore`

- [ ] **Step 1: Verify gitignore ignores Python build artifacts**

Add to `.gitignore` if not present:

```
ocr-server/.venv/
ocr-server/__pycache__/
ocr-server/ocr_server/__pycache__/
ocr-server/ocr_server/**/__pycache__/
ocr-server/*.egg-info/
ocr-server/*.egg-info/
```
Note: Generated proto files (`ocr_pb2.py`, `ocr_pb2_grpc.py`, `ocr.pb.go`, `ocr_grpc.pb.go`) must be committed — they are needed for Go/Python compilation.

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: update gitignore for OCR server artifacts"
```
