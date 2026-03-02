#!/usr/bin/env python3
"""Find cards in scryfall_cards.json.zst and print their JSON objects."""

import argparse
import json
from pathlib import Path

import zstandard


def load_scryfall_cards(path: Path) -> list[dict]:
    with open(path, "rb") as f:
        data = zstandard.decompress(f.read())
    return json.loads(data)


def main() -> None:
    parser = argparse.ArgumentParser(description="Find cards in scryfall_cards.json.zst")
    parser.add_argument("--name", "-n", help="Find cards matching this name (case-insensitive substring)")
    args = parser.parse_args()

    script_dir = Path(__file__).parent.parent
    cards_path = script_dir / "assets" / "card_info" / "scryfall_cards.json.zst"
    cards = load_scryfall_cards(cards_path)

    if not args.name:
        parser.print_help()
        return

    for card in cards:
        name = card.get("CardName", "")
        if args.name.lower() in name.lower():
            print(json.dumps(card, indent=2))


if __name__ == "__main__":
    main()
