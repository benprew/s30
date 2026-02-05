#!/usr/bin/env python3
"""Find cards with specific abilities in parsed_cards.json."""

import argparse
import json
from pathlib import Path


def load_parsed_cards(path: Path) -> list[dict]:
    with open(path) as f:
        return json.load(f)


def has_keyword(card: dict, keyword: str) -> bool:
    for ability in card.get("abilities", []):
        keywords = ability.get("Keywords") or []
        if keyword in keywords:
            return True
        effect = ability.get("Effect") or {}
        effect_keywords = effect.get("Keywords") or []
        if keyword in effect_keywords:
            return True
    return False


def has_ability_type(card: dict, ability_type: str) -> bool:
    for ability in card.get("abilities", []):
        if ability.get("Type") == ability_type:
            return True
    return False


def main() -> None:
    parser = argparse.ArgumentParser(description="Find cards with specific abilities")
    parser.add_argument("--keyword", "-k", help="Find cards with this keyword (e.g., Flying, Reach)")
    parser.add_argument("--type", "-t", help="Find cards with this ability type (e.g., Keyword, Activated, Triggered)")
    parser.add_argument("--list-keywords", action="store_true", help="List all unique keywords")
    parser.add_argument("--list-types", action="store_true", help="List all unique ability types")
    args = parser.parse_args()

    script_dir = Path(__file__).parent.parent
    parsed_cards_path = script_dir / "assets" / "card_info" / "parsed_cards.json"
    cards = load_parsed_cards(parsed_cards_path)

    if args.list_keywords:
        keywords: set[str] = set()
        for card in cards:
            for ability in card.get("abilities", []):
                for k in ability.get("Keywords") or []:
                    keywords.add(k)
                effect = ability.get("Effect") or {}
                for k in effect.get("Keywords") or []:
                    keywords.add(k)
        for k in sorted(keywords):
            print(k)
        return

    if args.list_types:
        types: set[str] = set()
        for card in cards:
            for ability in card.get("abilities", []):
                if t := ability.get("Type"):
                    types.add(t)
        for t in sorted(types):
            print(t)
        return

    matches: list[str] = []
    for card in cards:
        name = card.get("card_name", "Unknown")
        match = True
        if args.keyword and not has_keyword(card, args.keyword):
            match = False
        if args.type and not has_ability_type(card, args.type):
            match = False
        if match and (args.keyword or args.type):
            matches.append(name)

    for name in sorted(matches):
        print(name)


if __name__ == "__main__":
    main()
