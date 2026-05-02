from abc import ABC, abstractmethod


class OCREngine(ABC):
    """Abstract base for OCR engines."""

    @abstractmethod
    def extract_from_image(self, image_path: str) -> str:
        """Extract text from a single image file path. Returns the recognized text."""
        ...

    def extract_from_images(self, image_paths: list[str]) -> str:
        """Extract text from multiple images, concatenating results."""
        texts: list[str] = []
        for i, path in enumerate(image_paths, 1):
            t = self.extract_from_image(path)
            if t.strip():
                texts.append(f"[Página {i}]\n{t}")
        return "\n---\n".join(texts) if texts else ""
