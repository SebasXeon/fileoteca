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
