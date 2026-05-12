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
        pdf.close()
        return None
    page = pdf[0]
    bitmap = page.render(scale=1)
    pil_image = bitmap.to_pil()
    pdf.close()
    return _resize_and_save(pil_image)


def _thumbnail_from_image(file_path: str) -> str | None:
    try:
        with Image.open(file_path) as img:
            if img.mode in ("RGBA", "P", "LA"):
                img = img.convert("RGB")
            return _resize_and_save(img)
    except Exception:
        return None


def _resize_and_save(img: Image.Image) -> str:
    w, h = img.size
    stem = Path(img.filename).stem if hasattr(img, "filename") and img.filename else "img"
    if w > THUMBNAIL_MAX_WIDTH:
        ratio = THUMBNAIL_MAX_WIDTH / w
        new_h = int(h * ratio)
        img = img.resize((THUMBNAIL_MAX_WIDTH, new_h), Image.LANCZOS)
    out_path = str(Path(tempfile.gettempdir()) / f"thumb_{stem}.jpg")
    img.save(out_path, "JPEG", quality=75)
    return out_path
