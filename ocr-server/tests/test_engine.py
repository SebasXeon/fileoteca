import pytest
from ocr_server.engine.registry import create_engine, available_engines


def test_registry_has_winocr():
    import ocr_server.engine.winocr  # register
    engs = available_engines()
    assert "winocr" in engs


def test_unknown_engine_raises():
    with pytest.raises(ValueError, match="Unknown OCR engine"):
        create_engine("nonexistent")
