from concurrent.futures import ThreadPoolExecutor, Future


class OCRQueue:
    """Ensures only one OCR job runs at a time."""

    def __init__(self) -> None:
        self._executor = ThreadPoolExecutor(max_workers=1)

    def submit(self, fn, *args, **kwargs) -> Future:
        return self._executor.submit(fn, *args, **kwargs)

    def shutdown(self) -> None:
        self._executor.shutdown(wait=True)
