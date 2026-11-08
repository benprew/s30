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
    VersionExclusion,
    iter_json_records,
    normalize_type_line,
    parse_version_exclusion,
    process_card_records,
    save_json_and_zst,
    transform_raw_card,
)


class UpdateCardsJsonTest(unittest.TestCase):
    def test_art_sets_add_images_without_adding_playable_cards(self) -> None:
        records: list[dict[str, Any]] = [
            {
                "name": "Lightning Bolt",
                "set": "m10",
                "collector_number": "146",
                "image_uris": {"border_crop": "https://example.test/m10.jpg"},
            },
            {
                "name": "New Card",
                "set": "m10",
                "collector_number": "1",
                "image_uris": {"border_crop": "https://example.test/new.jpg"},
            },
            {
                "name": "Lightning Bolt",
                "set": "leb",
                "collector_number": "161",
                "image_uris": {"border_crop": "https://example.test/leb.jpg"},
            },
            {
                "name": "Lightning Bolt",
                "set": "4ed",
                "collector_number": "208",
                "image_uris": {"border_crop": "https://example.test/4ed.jpg"},
            },
            {
                "name": "Lightning Bolt",
                "set": "m10",
                "collector_number": "146",
                "lang": "de",
                "image_uris": {"border_crop": "https://example.test/german.jpg"},
            },
        ]
        cards = list(
            process_card_records(iter(records), ["leb", "4ed"], set(), art_sets=["m10"])
        )
        self.assertEqual(len(cards), 1)
        self.assertEqual(cards[0]["Artworks"][0]["SetID"], "leb")
        self.assertEqual(
            [art["SetID"] for art in cards[0]["Artworks"][1:]], ["4ed", "m10"]
        )
        self.assertEqual(
            cards[0]["Artworks"][1:][1]["BorderCropURL"], "https://example.test/m10.jpg"
        )
        reversed_cards = list(
            process_card_records(
                iter(reversed(records)), ["leb", "4ed"], set(), art_sets=["m10"]
            )
        )
        self.assertEqual(cards, reversed_cards)

    def test_alternate_art_survives_processing_transformed_data(self) -> None:
        record: dict[str, Any] = {
            "CardName": "Forest",
            "SetID": "leb",
            "CollectorNo": "300",
            "BorderCropURL": "https://example.test/leb.jpg",
            "AlternateArt": [
                {
                    "SetID": "unh",
                    "CollectorNo": "140",
                    "BorderCropURL": "https://example.test/unh.jpg",
                }
            ],
        }
        cards = list(process_card_records(iter([record]), ["leb"], set()))
        self.assertEqual(cards[0]["Artworks"][1:], record["AlternateArt"])

    def test_multiple_artworks_from_one_set_are_retained(self) -> None:
        records = [
            {
                "CardName": "Forest",
                "SetID": "leb",
                "CollectorNo": number,
                "BorderCropURL": f"https://example.test/{number}.jpg",
            }
            for number in ["300", "301", "302", "301"]
        ]
        cards = list(process_card_records(iter(records), ["leb"], set()))
        self.assertEqual(cards[0]["Artworks"][0]["CollectorNo"], "300")
        self.assertEqual(
            [art["CollectorNo"] for art in cards[0]["Artworks"][1:]], ["301", "302"]
        )

    def test_excluded_artwork_is_not_reintroduced(self) -> None:
        records = [
            {
                "CardName": "Forest",
                "SetID": set_id,
                "CollectorNo": "1",
                "BorderCropURL": f"https://example.test/{set_id}.jpg",
            }
            for set_id in ["leb", "4ed", "unh"]
        ]
        cards = list(
            process_card_records(
                iter(records),
                ["leb", "4ed"],
                set(),
                art_sets=["unh"],
                excluded_versions=[
                    VersionExclusion("Forest", "4ed"),
                    VersionExclusion("Forest", "unh"),
                ],
            )
        )
        self.assertEqual(len(cards[0]["Artworks"]), 1)

    def test_unified_card_json_keeps_urls_inside_artworks(self) -> None:
        records: list[dict[str, Any]] = [
            {
                "CardName": "Lightning Bolt",
                "ManaCost": "{R}",
                "Artworks": [
                    {
                        "SetID": "leb",
                        "CollectorNo": "162",
                        "PngURL": "https://example.test/leb.png",
                        "ArtURL": "https://example.test/leb-crop.jpg",
                    },
                    {
                        "SetID": "m10",
                        "CollectorNo": "146",
                        "BorderCropURL": "https://example.test/m10.jpg",
                    },
                ],
            }
        ]
        cards = list(process_card_records(iter(records), ["leb"], set()))
        self.assertEqual(cards, records)
        self.assertNotIn("SetID", cards[0])
        self.assertNotIn("PngURL", cards[0])
        self.assertEqual(
            cards[0]["Artworks"][0]["ArtURL"], "https://example.test/leb-crop.jpg"
        )

    def test_unified_card_json_skips_card_when_all_artworks_excluded(self) -> None:
        records: list[dict[str, Any]] = [
            {
                "CardName": "Lightning Bolt",
                "ManaCost": "{R}",
                "Artworks": [
                    {
                        "SetID": "leb",
                        "CollectorNo": "162",
                        "PngURL": "https://example.test/leb.png",
                    }
                ],
            }
        ]
        cards = list(
            process_card_records(
                iter(records),
                ["leb"],
                set(),
                excluded_versions=[VersionExclusion("Lightning Bolt", "leb")],
            )
        )
        self.assertEqual(cards, [])

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

    def test_normalize_type_line(self) -> None:
        self.assertEqual(normalize_type_line("Summon Knights"), "Creature Knights")
        self.assertEqual(normalize_type_line("Summon Dragon"), "Creature Dragon")
        self.assertEqual(normalize_type_line("Summon Jaguar"), "Creature Jaguar")
        self.assertEqual(normalize_type_line("Summon Goblin"), "Creature Goblin")
        self.assertEqual(normalize_type_line("Summon"), "Creature")
        self.assertEqual(
            normalize_type_line("Creature — Human Soldier"),
            "Creature — Human Soldier",
        )
        self.assertEqual(normalize_type_line("Instant"), "Instant")
        self.assertEqual(normalize_type_line("Artifact"), "Artifact")
        self.assertIsNone(normalize_type_line(None))

    def test_transform_raw_card_astral_creature(self) -> None:
        raw: dict[str, Any] = {
            "name": "Rainbow Knights",
            "mana_cost": "{W}{W}",
            "type_line": "Summon Knights",
            "set": "past",
            "collector_number": "9",
        }
        transformed = transform_raw_card(raw)
        self.assertEqual(transformed["CardName"], "Rainbow Knights")
        self.assertEqual(transformed["TypeLine"], "Creature Knights")

    def test_process_card_records_normalizes_existing_astral_cards(self) -> None:
        records: list[dict[str, Any]] = [
            {
                "CardName": "Rainbow Knights",
                "SetID": "past",
                "TypeLine": "Summon Knights",
            }
        ]
        processed = list(
            process_card_records(
                records=iter(records),
                allowed_sets=DEFAULT_ALLOWED_SETS,
                excluded_names=DEFAULT_EXCLUDED_NAMES,
            )
        )
        self.assertEqual(len(processed), 1)
        self.assertEqual(processed[0]["TypeLine"], "Creature Knights")

    def test_process_card_records_filtering(self) -> None:
        records: list[dict[str, Any]] = [
            # Included: Black Lotus (leb)
            {
                "name": "Black Lotus",
                "set": "leb",
                "collector_number": "233",
                "lang": "en",
            },
            # Excluded by name: Chaos Orb
            {
                "name": "Chaos Orb",
                "set": "leb",
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
        )

        names_and_sets = [(c["CardName"], c["Artworks"][0]["SetID"]) for c in processed]
        self.assertEqual(
            names_and_sets,
            [("Black Lotus", "leb"), ("El-Hajjâj", "arn")],
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
