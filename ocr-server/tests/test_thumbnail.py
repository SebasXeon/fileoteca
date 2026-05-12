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
        with Image.open(result) as thumb:
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
        with Image.open(result) as thumb:
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
        with Image.open(result) as thumb:
            assert thumb.width <= 400
        os.remove(result)
    finally:
        tmp_pdf.unlink(missing_ok=True)
