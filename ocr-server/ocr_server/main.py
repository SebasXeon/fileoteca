import argparse
import logging
import sys
from concurrent import futures
from pathlib import Path

import grpc
import tomli

from ocr_server.proto import ocr_pb2, ocr_pb2_grpc
from ocr_server.pipeline import pdf as pdf_pipeline
from ocr_server.pipeline import image as image_pipeline
from ocr_server.queue import OCRQueue

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("ocr-server")

IMAGE_EXTS = {"png", "jpg", "jpeg", "gif", "bmp", "svg", "webp", "tiff", "ico"}
PDF_EXTS = {"pdf"}


def load_config() -> dict:
    config_path = Path(__file__).parent.parent / "config.toml"
    if not config_path.exists():
        return {"engine": "winocr", "host": "[::1]", "port": 50051}
    with open(config_path, "rb") as f:
        return tomli.load(f)


class OCRService(ocr_pb2_grpc.OCREngineServicer):
    def __init__(self, engine_name: str, queue: OCRQueue) -> None:
        self._engine = engine_name
        self._queue = queue

    def ExtractText(self, request, context):
        logger.info("Processing document %s (%s)", request.id, request.file_type)
        try:
            future = self._queue.submit(
                _extract_text, request.id, request.file_path, request.file_type, self._engine
            )
            text = future.result(timeout=300)
            return ocr_pb2.ExtractResponse(text=text)
        except Exception as exc:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(exc))
            logger.error("OCR failed for %s: %s", request.id, exc)
            return ocr_pb2.ExtractResponse(text="")


def _extract_text(doc_id: str, file_path: str, file_type: str, engine: str) -> str:
    ext = file_type.lower().lstrip(".")

    if ext in PDF_EXTS:
        logger.info("Rendering PDF pages for %s", doc_id)
        image_paths = pdf_pipeline.pdf_to_images(file_path)
        if not image_paths:
            return ""
        result = image_pipeline.extract_text(image_paths, engine)
        for p in image_paths:
            Path(p).unlink(missing_ok=True)
        return result

    if ext in IMAGE_EXTS:
        return image_pipeline.extract_text([file_path], engine)

    # Office / unknown — lazy-import markitdown (avoids onnxruntime at startup)
    from ocr_server.pipeline import markitdown_
    return markitdown_.extract_text(file_path)


def run_server(config: dict | None = None) -> None:
    if config is None:
        config = load_config()

    host = config.get("host", "[::1]")
    port = config.get("port", 50051)
    engine_name = config.get("engine", "winocr")
    address = f"{host}:{port}"

    queue = OCRQueue()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    ocr_pb2_grpc.add_OCREngineServicer_to_server(OCRService(engine_name, queue), server)
    server.add_insecure_port(address)
    server.start()
    logger.info("OCR server started on %s (engine: %s)", address, engine_name)
    server.wait_for_termination()
    queue.shutdown()


def run_cli(file_path: str, config: dict | None = None) -> None:
    if config is None:
        config = load_config()
    engine_name = config.get("engine", "winocr")
    file_type = Path(file_path).suffix
    text = _extract_text("cli", file_path, file_type, engine_name)
    if text.strip():
        print(text)
    else:
        print("(no text extracted)", file=sys.stderr)
        sys.exit(1)


def main() -> None:
    parser = argparse.ArgumentParser(description="OCR gRPC server")
    sub = parser.add_subparsers(dest="command")

    server_parser = sub.add_parser("server", help="Start gRPC server")
    server_parser.add_argument("--config", type=str, default=None, help="Path to config.toml")

    ocr_parser = sub.add_parser("ocr", help="One-shot OCR on a file")
    ocr_parser.add_argument("file", type=str, help="Path to file")
    ocr_parser.add_argument("--config", type=str, default=None, help="Path to config.toml")

    args = parser.parse_args()

    config = {}
    if hasattr(args, "config") and args.config:
        with open(args.config, "rb") as f:
            config = tomli.load(f)

    if args.command == "ocr":
        run_cli(args.file, config)
    else:
        run_server(config)


if __name__ == "__main__":
    main()
