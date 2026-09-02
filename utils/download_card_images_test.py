import tempfile
import unittest
import zipfile
from pathlib import Path

from PIL import Image

from utils.download_card_images import (
    CARD_WIDTH,
    build_card_image,
    download_and_process_card,
    get_frame_filename,
    retain_archive_files,
)


class DownloadCardImagesTest(unittest.TestCase):
    def setUp(self) -> None:
        self.assets_dir = Path("assets")

    def test_get_frame_filename(self) -> None:
        self.assertEqual(
            get_frame_filename({"Colors": ["W"], "TypeLine": "Creature — Angel"}),
            "Cardbk_White.pic.png",
        )
        self.assertEqual(
            get_frame_filename({"Colors": ["U"], "TypeLine": "Instant"}),
            "Cardbk_Blue.pic.png",
        )
        self.assertEqual(
            get_frame_filename({"Colors": ["B"], "TypeLine": "Sorcery"}),
            "Cardbk_Black.pic.png",
        )
        self.assertEqual(
            get_frame_filename({"Colors": ["R"], "TypeLine": "Instant"}),
            "Cardbk_Red.pic.png",
        )
        self.assertEqual(
            get_frame_filename({"Colors": ["G"], "TypeLine": "Creature — Elf"}),
            "Cardbk_Green.pic.png",
        )
        self.assertEqual(
            get_frame_filename({"Colors": ["W", "U"], "TypeLine": "Creature"}),
            "Cardbk_Gold.pic.png",
        )
        self.assertEqual(
            get_frame_filename(
                {"Colors": [], "TypeLine": "Artifact Creature — Construct"}
            ),
            "Cardbk_Artifact.pic.png",
        )
        self.assertEqual(
            get_frame_filename({"Colors": [], "TypeLine": "Basic Land — Mountain"}),
            "Cardbk_Redland.pic.png",
        )
        self.assertEqual(
            get_frame_filename({"Colors": [], "TypeLine": "Land", "SetID": "arn"}),
            "Cardbk_Arabiannightsland.pic.png",
        )
        self.assertEqual(
            get_frame_filename({"Colors": [], "TypeLine": "Land", "SetID": "atq"}),
            "Cardbk_Antiquitiesland.pic.png",
        )

    def test_build_card_image(self) -> None:
        art = Image.new("RGB", (300, 200), (50, 100, 150))
        card_data = {
            "CardName": "Serra Angel",
            "ManaCost": "{3}{W}{W}",
            "Colors": ["W"],
            "TypeLine": "Creature — Angel",
            "Text": "Flying\nVigilance",
            "Power": "4",
            "Toughness": "4",
            "FlavorText": "Born with wings of light.",
            "Artist": "Douglas Shuler",
            "SetID": "2ed",
        }
        card_img = build_card_image(card_data, art, self.assets_dir)
        self.assertEqual(card_img.size, (228, 325))
        self.assertEqual(card_img.mode, "RGBA")

    def test_download_and_process_card_builds_card_image(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            source = root / "art.jpg"
            output = root / "output"
            output.mkdir()
            Image.new("RGB", (400, 300), "red").save(source, format="JPEG")

            success, filename = download_and_process_card(
                {
                    "CardName": "Test Creature",
                    "SetID": "tst",
                    "CollectorNo": "1",
                    "ManaCost": "{1}{R}",
                    "Colors": ["R"],
                    "TypeLine": "Creature — Goblin",
                    "Text": "{T}: Deal 1 damage.",
                    "Power": "1",
                    "Toughness": "1",
                    "Artist": "Test Artist",
                    "ArtURL": source.as_uri(),
                },
                set(),
                output,
                self.assets_dir,
            )

            self.assertTrue(success)
            self.assertEqual(filename, "tst-1-200-test-creature.jpg")
            with Image.open(output / filename) as processed:
                self.assertEqual(processed.format, "JPEG")
                self.assertEqual(processed.width, CARD_WIDTH)
                self.assertEqual(processed.height, 349)

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
