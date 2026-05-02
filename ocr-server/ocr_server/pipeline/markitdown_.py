from markitdown import MarkItDown


def extract_text(file_path: str) -> str:
    """Extract plain text from an Office document using MarkItDown."""
    md = MarkItDown()
    result = md.convert(file_path)
    return result.text_content if result else ""
