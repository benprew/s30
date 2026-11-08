import json
import tempfile
import unittest
import zipfile
from contextlib import chdir
from pathlib import Path
from unittest.mock import patch

import zstandard
from PIL import Image

from utils.download_card_images import (
    CARD_WIDTH,
    download_and_process_card,
    main,
    retain_archive_files,
)


class DownloadCardImagesTest(unittest.TestCase):
    def test_failed_alternate_download_does_not_prune_archive(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            archive = root / "assets/art/cardimages.zip"
            archive.parent.mkdir(parents=True)
            with zipfile.ZipFile(archive, "w") as images:
                images.writestr("leb-161-200-lightning-bolt.jpg", b"existing image")
                images.writestr("previous-art.jpg", b"keep until update succeeds")
            source = root / "cards.json"
            source.write_text(
                json.dumps(
                    [
                        {
                            "CardName": "Lightning Bolt",
                            "SetID": "leb",
                            "CollectorNo": "161",
                            "BorderCropURL": (root / "default.jpg").as_uri(),
                            "AlternateArt": [
                                {
                                    "SetID": "m10",
                                    "CollectorNo": "146",
                                    "PngURL": (root / "missing.png").as_uri(),
                                }
                            ],
                        }
                    ]
                )
            )
            with (
                chdir(root),
                patch("sys.argv", ["download_card_images.py", str(source)]),
            ):
                with self.assertRaises(SystemExit), self.assertLogs(level="ERROR"):
                    main()
            with zipfile.ZipFile(archive) as images:
                self.assertEqual(
                    set(images.namelist()),
                    {"leb-161-200-lightning-bolt.jpg", "previous-art.jpg"},
                )

    def test_png_url_is_used_when_border_crop_is_missing(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            source = root / "source.png"
            Image.new("RGB", (480, 680), "blue").save(source)
            success, filename = download_and_process_card(
                {
                    "CardName": "Forest",
                    "SetID": "unh",
                    "CollectorNo": "140",
                    "PngURL": source.as_uri(),
                },
                set(),
                root,
            )
            self.assertTrue(success)
            self.assertEqual(filename, "unh-140-200-forest.jpg")
            with Image.open(root / filename) as processed:
                self.assertEqual(processed.width, CARD_WIDTH)

    def test_cli_downloads_and_retains_every_artwork(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            default = root / "default.jpg"
            alternate = root / "alternate.jpg"
            Image.new("RGB", (245, 342), "red").save(default)
            Image.new("RGB", (245, 342), "blue").save(alternate)
            data = [
                {
                    "CardName": "Lightning Bolt",
                    "Artworks": [
                        {
                            "SetID": "leb",
                            "CollectorNo": "161",
                            "BorderCropURL": default.as_uri(),
                        },
                        {
                            "SetID": "m10",
                            "CollectorNo": "146",
                            "BorderCropURL": alternate.as_uri(),
                        },
                        {
                            "SetID": "m10",
                            "CollectorNo": "146",
                            "BorderCropURL": alternate.as_uri(),
                        },
                    ],
                }
            ]
            source = root / "cards.json"
            source.write_text(json.dumps(data))
            with (
                chdir(root),
                patch("sys.argv", ["download_card_images.py", str(source)]),
            ):
                main()
                archive = root / "assets/art/cardimages.zip"
                expected = {
                    "leb-161-200-lightning-bolt.jpg",
                    "m10-146-200-lightning-bolt.jpg",
                }
                with zipfile.ZipFile(archive) as images:
                    self.assertEqual(set(images.namelist()), expected)
                    with images.open("m10-146-200-lightning-bolt.jpg") as image_file:
                        with Image.open(image_file) as image:
                            self.assertEqual(image.width, CARD_WIDTH)
                            pixel = image.getpixel((0, 0))
                            assert isinstance(pixel, tuple)
                            self.assertGreater(pixel[2], 200)
                compressed_source = root / "cards.json.zst"
                compressed_source.write_bytes(
                    zstandard.ZstdCompressor().compress(source.read_bytes())
                )
                with (
                    patch(
                        "sys.argv", ["download_card_images.py", str(compressed_source)]
                    ),
                    patch("urllib.request.urlopen") as fetch,
                ):
                    main()
                    fetch.assert_not_called()
                with zipfile.ZipFile(archive) as images:
                    self.assertEqual(set(images.namelist()), expected)

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
