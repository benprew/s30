import tempfile
import unittest
from pathlib import Path

from PIL import Image

from utils.download_card_images import CARD_WIDTH, download_and_process_card


class DownloadCardImagesTest(unittest.TestCase):
    def test_download_and_process_card_resizes_local_image(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            source = root / "source.png"
            output = root / "output"
            output.mkdir()
            Image.new("RGB", (490, 684), "red").save(source)

            success, filename = download_and_process_card(
                {
                    "CardName": "Test Card",
                    "SetID": "tst",
                    "CollectorNo": "1",
                    "PngURL": source.as_uri(),
                },
                set(),
                output,
            )

            self.assertTrue(success)
            self.assertEqual(filename, "tst-1-200-test-card.png")
            with Image.open(output / filename) as processed:
                self.assertEqual(processed.width, CARD_WIDTH)
                self.assertEqual(processed.height, 342)


if __name__ == "__main__":
    unittest.main()
