from ..engine import create_engine


def extract_text(image_paths: list[str], engine_name: str) -> str:
    """Extract text from a list of image paths using the specified OCR engine."""
    engine = create_engine(engine_name)
    return engine.extract_from_images(image_paths)
