import tempfile
from pathlib import Path
import pypdfium2 as pdfium

HEAD_PAGES = 5
TAIL_PAGES = 2


def pdf_to_images(file_path: str, temp_dir: str | None = None) -> list[str]:
    """Render selected pages of a PDF to temporary PNG images."""
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
