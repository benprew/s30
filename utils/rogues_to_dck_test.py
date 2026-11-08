import unittest

from utils.rogues_to_dck import build_card_index, format_dck_line, pick_printing


class RogueDeckArtworkTest(unittest.TestCase):
    def test_export_uses_printing_metadata_from_artworks(self) -> None:
        cards = [
            {
                "CardName": "Lightning Bolt",
                "Artworks": [
                    {"SetID": "leb", "CollectorNo": "162"},
                    {"SetID": "m10", "CollectorNo": "146"},
                ],
            }
        ]
        printings = build_card_index(cards)["lightning bolt"]
        self.assertEqual(len(printings), 2)
        self.assertEqual(
            format_dck_line("4", pick_printing(printings), "Lightning Bolt", False),
            "4 [LEB:162] Lightning Bolt",
        )

    def test_export_accepts_legacy_flat_printings(self) -> None:
        cards = [{"CardName": "Forest", "SetID": "leb", "CollectorNo": "300"}]
        self.assertEqual(build_card_index(cards)["forest"][0], cards[0])
