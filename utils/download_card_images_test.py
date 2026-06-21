import tempfile
import unittest
import zipfile
from pathlib import Path

from PIL import Image

from utils.download_card_images import (
    CARD_WIDTH,
    download_and_process_card,
    retain_archive_files,
)


class DownloadCardImagesTest(unittest.TestCase):
    def test_download_and_process_card_resizes_local_image(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            source = root / "source.jpg"
            output = root / "output"
            output.mkdir()
            Image.new("RGB", (480, 680), "red").save(source, format="JPEG")

            success, filename = download_and_process_card(
                {
                    "CardName": "Test Card",
                    "SetID": "tst",
                    "CollectorNo": "1",
                    "BorderCropURL": source.as_uri(),
                },
                set(),
                output,
            )

            self.assertTrue(success)
            self.assertEqual(filename, "tst-1-200-test-card.jpg")
            with Image.open(output / filename) as processed:
                self.assertEqual(processed.format, "JPEG")
                self.assertEqual(processed.width, CARD_WIDTH)
                self.assertEqual(processed.height, 347)

    def test_retain_archive_files_removes_old_png_images(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            archive = Path(temp_dir) / "cards.zip"
            with zipfile.ZipFile(archive, "w") as cards:
                cards.writestr("old-card.png", b"old")
                cards.writestr("new-card.jpg", b"new")

            retain_archive_files(archive, {"new-card.jpg"})

            with zipfile.ZipFile(archive) as cards:
                self.assertEqual(cards.namelist(), ["new-card.jpg"])
                self.assertEqual(cards.read("new-card.jpg"), b"new")


if __name__ == "__main__":
    unittest.main()
