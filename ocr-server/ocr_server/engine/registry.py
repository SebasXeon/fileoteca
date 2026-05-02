from .base import OCREngine

_engines: dict[str, type[OCREngine]] = {}


def register(name: str):
    def decorator(cls: type[OCREngine]) -> type[OCREngine]:
        _engines[name] = cls
        return cls
    return decorator


def create_engine(name: str) -> OCREngine:
    if name not in _engines:
        raise ValueError(f"Unknown OCR engine: {name!r}. Available: {list(_engines.keys())}")
    return _engines[name]()


def available_engines() -> list[str]:
    return list(_engines.keys())
