# ocr_winrt_example.py
import asyncio
from PIL import Image
import winrt.windows.storage.streams as streams
from winrt.windows.media.ocr import OcrEngine
from winrt.windows.graphics.imaging import SoftwareBitmap, BitmapPixelFormat
import platform

def check_windows_version(min_major=10, min_build=17763):  
    """
    Verifica que el SO sea Windows >= 10 (1809 / build 17763).
    """
    if platform.system() != "Windows":
        return False, "Este OCR solo funciona en Windows"

    version = platform.version().split(".")
    if len(version) >= 3:
        major, build = int(version[0]), int(version[2])
        if major < min_major or build < min_build:
            return False, f"Windows demasiado antiguo (build {major}.{build})"
    return True, "Versión de Windows compatible"

def check_ocr_engine():
    """
    Verifica que OcrEngine pueda inicializarse.
    """
    engine = OcrEngine.try_create_from_user_profile_languages()
    if engine is None:
        return False, "No se pudo inicializar OcrEngine (quizá faltan paquetes de idioma OCR)"
    return True, f"OcrEngine disponible"

async def verify_environment():
    ok, msg = check_windows_version()
    print("[Windows]", msg)
    if not ok:
        return False

    ok, msg = check_ocr_engine()
    print("[OcrEngine]", msg)
    return ok

def pil_to_software_bitmap(path):
    img = Image.open(path).convert("RGBA")
    writer = streams.DataWriter()
    writer.write_bytes(img.tobytes())
    bitmap = SoftwareBitmap(BitmapPixelFormat.RGBA8, img.width, img.height)
    bitmap.copy_from_buffer(writer.detach_buffer())
    return bitmap

async def recognize_async(image_path):
    bitmap = pil_to_software_bitmap(image_path)
    engine = OcrEngine.try_create_from_user_profile_languages()
    if engine is None:
        raise RuntimeError("No OCR engine for user languages available")
    result = await engine.recognize_async(bitmap)
    return result.text

if __name__ == "__main__":
    success = asyncio.run(verify_environment())
    if success:
        print("✅ Entorno listo para usar Windows.Media.Ocr")
    else:
        print("⚠️  OCR nativo no disponible, usar fallback (ej. Tesseract)")

    text = asyncio.run(recognize_async("C:/Users/Sebas/Pictures/evidencia 1.png"))
    print("\n"+text)