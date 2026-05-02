import asyncio
from PIL import Image
import winrt.windows.storage.streams as streams
from winrt.windows.media.ocr import OcrEngine
from winrt.windows.graphics.imaging import SoftwareBitmap, BitmapPixelFormat

from .base import OCREngine
from .registry import register


def _pil_to_software_bitmap(path: str) -> SoftwareBitmap:
    img = Image.open(path).convert("RGBA")
    writer = streams.DataWriter()
    writer.write_bytes(img.tobytes())
    bitmap = SoftwareBitmap(BitmapPixelFormat.RGBA8, img.width, img.height)
    bitmap.copy_from_buffer(writer.detach_buffer())
    return bitmap


@register("winocr")
class WinOCREngine(OCREngine):
    def extract_from_image(self, image_path: str) -> str:
        bitmap = _pil_to_software_bitmap(image_path)
        engine = OcrEngine.try_create_from_user_profile_languages()
        if engine is None:
            raise RuntimeError("No OCR engine available for this language")
        result = asyncio.run(engine.recognize_async(bitmap))
        return result.text if result else ""
