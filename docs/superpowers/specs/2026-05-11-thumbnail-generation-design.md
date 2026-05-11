# Thumbnail Generation — Design

**Date**: 2026-05-11  
**Status**: Approved

## Summary

Extend the OCR server into a document-tools server by adding thumbnail generation. When a document is processed, the Python server generates a small JPEG thumbnail (max 400px width). The Go backend uploads it to a new `thumbnail` file field on the `documents` PocketBase collection. The frontend displays the thumbnail in file cards and the document detail view.

## Requirements

| Requirement | Detail |
|---|---|
| Scope | All document types (PDF, images, office) |
| Format | JPEG, max 400px width, compressed |
| Storage | PocketBase `thumbnail` file field on `documents` |
| Timing | Separate gRPC call after OCR completes |

## Architecture

```
Go backend (PocketBase)                 Python server (gRPC)
     │                                        │
     ├── OCRWorker.processJob()               │
     │   ├── ExtractText() ──────────────────>│  PDF → render pages 1-5 + tail → OCR → text
     │   │   <────── text ────────────────────│
     │   │                                    │
     │   ├── GenerateThumbnail() ─────────────>│  PDF → render page 1 → resize → JPEG
     │   │   <──── temp file path ────────────│  Image → resize → JPEG
     │   │                                    │  Office → return None
     │   ├── read temp file                   │
     │   ├── upload to PB `thumbnail` field   │
     │   └── cleanup temp file                │
     │                                        │
     └── Frontend reads thumbnail URL from PB API
```

## Changes by layer

### 1. Proto (`ocr-server/proto/ocr.proto`)

Add messages and RPC:

```protobuf
message ThumbnailRequest {
    string id = 1;
    string file_path = 2;
    string file_type = 3;
}

message ThumbnailResponse {
    string thumbnail_path = 1;
}

service OCREngine {
    rpc ExtractText(ExtractRequest) returns (ExtractResponse);
    rpc GenerateThumbnail(ThumbnailRequest) returns (ThumbnailResponse);
}
```

Regenerate Python pb2 files and Go pb files.

### 2. Python — new pipeline module (`ocr_server/pipeline/thumbnail.py`)

Single function:

- `generate_thumbnail(file_path, file_type) -> str | None`

Logic:
- **PDF**: pypdfium2 renders page 1 at scale=1, PIL resizes to max 400px width, saves JPEG to temp dir, returns path.
- **Images**: PIL opens, resizes to max 400px width, saves JPEG, returns path.
- **Office/unknown**: Returns `None`.

### 3. Python — gRPC handler (`ocr_server/main.py`)

Add handler to `OCRService`:

```python
def GenerateThumbnail(self, request, context):
    path = thumbnail.generate_thumbnail(request.file_path, request.file_type)
    return ocr_pb2.ThumbnailResponse(thumbnail_path=path or "")
```

### 4. Go — proto regeneration

Regenerate `internal/ocr/proto/ocr.pb.go` and `ocr_grpc.pb.go` from updated `.proto`.

### 5. Go — migration

Add `thumbnail` file field to `documents` collection:

```go
addField(documents, &core.FileField{Name: "thumbnail", MaxSelect: 1})
```

Add a new migration file `1776197415_add_thumbnail.go` (non-destructive additive migration).

### 6. Go — OCR client (`internal/ocr/client.go`)

New method:

```go
func (c *OcrClient) GenerateThumbnail(ctx context.Context, id, filePath, fileType string) (string, error)
```

Calls the new gRPC method, returns the temp file path.

### 7. Go — OCR worker (`internal/ocr/queue.go`)

In `processJob()`, after `updateDocumentStatus()` succeeds:

1. Call `GenerateThumbnail()`.
2. If path returned, read the temp file, upload to PocketBase via record's file field, save.
3. Clean up temp file (`os.Remove`).

### 8. Frontend — types (`web/src/lib/types.ts`)

Add `thumbnail?: string` to `ExplorerFile`.

### 9. Frontend — API (`web/src/lib/api/documents.ts`)

In `toExplorerFile()`, include thumbnail URL:

```ts
thumbnail: record.thumbnail ? pb.files.getURL(record, record.thumbnail) : undefined,
```

Also add `thumbnail` to `DocumentDetail`.

### 10. Frontend — file card (`web/src/lib/components/explorer/file-card.svelte`)

Add conditional thumbnail display: if `file.thumbnail` exists, show `<img>` instead of the file-type icon. Fallback to existing icon behavior.

### 11. Frontend — document detail (`web/src/routes/document/[id]/+page.svelte`)

Show thumbnail above metadata in the info card when available.

## Error handling

- Thumbnail generation failure is non-fatal — document processing continues without thumbnail.
- Office docs that can't be thumbnailed return `None`, frontend shows icon fallback.
- Go worker logs thumbnail errors but does not change document status.

## Testing

- Python: unit test `test_thumbnail.py` for PDF and image thumbnail generation.
- Go: test OCR worker calls thumbnail and uploads correctly (requires mock gRPC).
- Manual: add a PDF, verify thumbnail appears in frontend file card and detail view.
