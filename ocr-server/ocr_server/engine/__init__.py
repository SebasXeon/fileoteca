from .registry import create_engine, available_engines
from .base import OCREngine

from . import winocr  # registers @register("winocr")

__all__ = ["OCREngine", "create_engine", "available_engines"]
