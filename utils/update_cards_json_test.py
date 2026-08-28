import io
import json
import tempfile
import unittest
from pathlib import Path
from typing import Any

import zstandard

from utils.update_cards_json import (
    DEFAULT_ALLOWED_SETS,
    DEFAULT_EXCLUDED_NAMES,
    DEFAULT_EXCLUDED_VERSIONS,
    VersionExclusion,
    is_version_excluded,
    iter_json_records,
    parse_version_exclusion,
    process_card_records,
    save_json_and_zst,
    transform_raw_card,
)


class UpdateCardsJsonTest(unittest.TestCase):
    def test_parse_version_exclusion(self) -> None:
        self.assertEqual(
            parse_version_exclusion("El-Hajjâj:4ed"),
            VersionExclusion(name="El-Hajjâj", set_id="4ed", collector_number=None),
        )
        self.assertEqual(
            parse_version_exclusion("Drudge Skeletons:4ed:107†"),
            VersionExclusion(
                name="Drudge Skeletons", set_id="4ed", collector_number="107†"
            ),
        )
        self.assertEqual(
            parse_version_exclusion("El-Hajjâj::134†"),
            VersionExclusion(name="El-Hajjâj", set_id=None, collector_number="134†"),
        )
        self.assertEqual(
            parse_version_exclusion("Chaos Orb"),
            VersionExclusion(name="Chaos Orb", set_id=None, collector_number=None),
        )

    def test_is_version_excluded(self) -> None:
        exclusions = [
            VersionExclusion(name="El-Hajjâj", set_id="4ed"),
            VersionExclusion(
                name="Drudge Skeletons", set_id="4ed", collector_number="107†"
            ),
        ]
        # El-Hajjaj 4ed any collector number should be excluded
        self.assertTrue(is_version_excluded("El-Hajjâj", "4ed", "134", exclusions))
        self.assertTrue(is_version_excluded("El-Hajjâj", "4ed", "134†", exclusions))
        # El-Hajjaj arn should NOT be excluded
        self.assertFalse(is_version_excluded("El-Hajjâj", "arn", "24", exclusions))

        # Drudge Skeletons 107† in 4ed should be excluded
        self.assertTrue(
            is_version_excluded("Drudge Skeletons", "4ed", "107†", exclusions)
        )
        # Drudge Skeletons 133 in 4ed should NOT be excluded
        self.assertFalse(
            is_version_excluded("Drudge Skeletons", "4ed", "133", exclusions)
        )

    def test_transform_raw_card(self) -> None:
        raw: dict[str, Any] = {
            "name": "Black Lotus",
            "mana_cost": "{0}",
            "colors": [],
            "color_identity": [],
            "keywords": [],
            "type_line": "Artifact",
            "oracle_text": (
                "{T}, Sacrifice this artifact: Add three mana of any one color."
            ),
            "power": None,
            "toughness": None,
            "set_name": "Unlimited Edition",
            "set": "2ed",
            "collector_number": "233",
            "rarity": "rare",
            "frame": "1993",
            "flavor_text": None,
            "frame_effects": None,
            "watermark": None,
            "artist": "Christopher Rush",
            "produced_mana": ["B", "G", "R", "U", "W"],
            "prices": {"usd": "10000.00", "eur": "9000.00"},
            "legalities": {"vintage": "restricted"},
            "image_uris": {
                "png": "https://cards.scryfall.io/png/front/4/a/4a2e428c.png",
                "art_crop": "https://cards.scryfall.io/art_crop/front/4/a/4a2e428c.jpg",
                "border_crop": "https://cards.scryfall.io/border_crop/front/4/a/4a2e428c.jpg",
            },
        }

        transformed = transform_raw_card(raw)
        self.assertEqual(transformed["CardName"], "Black Lotus")
        self.assertEqual(transformed["SetID"], "2ed")
        self.assertEqual(transformed["CollectorNo"], "233")
        self.assertEqual(transformed["PriceUSD"], "10000.00")
        self.assertTrue(transformed["VintageRestricted"])
        self.assertEqual(
            transformed["BorderCropURL"],
            "https://cards.scryfall.io/border_crop/front/4/a/4a2e428c.jpg",
        )

    def test_process_card_records_filtering(self) -> None:
        records: list[dict[str, Any]] = [
            # Included: Black Lotus (2ed)
            {
                "name": "Black Lotus",
                "set": "2ed",
                "collector_number": "233",
                "lang": "en",
            },
            # Excluded by name: Chaos Orb
            {
                "name": "Chaos Orb",
                "set": "2ed",
                "collector_number": "236",
                "lang": "en",
            },
            # Excluded by set: blb
            {"name": "Forest", "set": "blb", "collector_number": "280", "lang": "en"},
            # Excluded by language (German)
            {
                "name": "El-Hajjâj",
                "set": "4ed",
                "collector_number": "134†",
                "lang": "de",
            },
            # Excluded by version: El-Hajjâj 4ed
            {
                "name": "El-Hajjâj",
                "set": "4ed",
                "collector_number": "134",
                "lang": "en",
            },
            # Included: El-Hajjâj arn
            {"name": "El-Hajjâj", "set": "arn", "collector_number": "24", "lang": "en"},
        ]

        processed = process_card_records(
            records=iter(records),
            allowed_sets=DEFAULT_ALLOWED_SETS,
            excluded_names=DEFAULT_EXCLUDED_NAMES,
            excluded_versions=DEFAULT_EXCLUDED_VERSIONS,
            allow_foreign=False,
        )

        names_and_sets = [(c["CardName"], c["SetID"]) for c in processed]
        self.assertEqual(
            names_and_sets,
            [("Black Lotus", "2ed"), ("El-Hajjâj", "arn")],
        )

    def test_iter_json_records_array_and_jsonl(self) -> None:
        # Array input
        array_data = json.dumps([{"name": "Card A"}, {"name": "Card B"}]).encode(
            "utf-8"
        )
        records_array = list(iter_json_records(io.BytesIO(array_data)))
        self.assertEqual(len(records_array), 2)
        self.assertEqual(records_array[0]["name"], "Card A")

        # JSONL input
        jsonl_data = b'{"name": "Card C"}\n{"name": "Card D"}\n'
        records_jsonl = list(iter_json_records(io.BytesIO(jsonl_data)))
        self.assertEqual(len(records_jsonl), 2)
        self.assertEqual(records_jsonl[1]["name"], "Card D")

    def test_save_json_and_zst(self) -> None:
        cards: list[dict[str, Any]] = [
            {"CardName": "Test Card", "SetID": "2ed", "CollectorNo": "1"}
        ]
        with tempfile.TemporaryDirectory() as tmpdir:
            json_path = Path(tmpdir) / "cards.json"
            zst_path = Path(tmpdir) / "cards.json.zst"

            save_json_and_zst(cards, json_path, zst_path)

            self.assertTrue(json_path.exists())
            self.assertTrue(zst_path.exists())

            with open(json_path, "r", encoding="utf-8") as f:
                loaded_json = json.load(f)
            self.assertEqual(loaded_json, cards)

            with open(zst_path, "rb") as f:
                dctx = zstandard.ZstdDecompressor()
                decompressed = dctx.decompress(f.read())
                loaded_zst = json.loads(decompressed.decode("utf-8"))
            self.assertEqual(loaded_zst, cards)


if __name__ == "__main__":
    unittest.main()
