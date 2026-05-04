#!/usr/bin/env python3
"""Convert rogue deck TOML configs in assets/configs/rogues/ to XMage .dck files.

Reads each .toml file, looks up each card in scryfall_cards.json.zst to find a
SetID and CollectorNo, and writes a .dck file with the XMage format:

    NAME:<deck name>
    <count> [<SET>:<COLLECTOR>] <Card Name>
    SB: <count> [<SET>:<COLLECTOR>] <Card Name>
"""

import argparse
import json
import sys
import tomllib
import unicodedata
from collections import defaultdict
from pathlib import Path

import zstandard


def load_scryfall_cards(path: Path) -> list[dict]:
    with open(path, "rb") as f:
        data = zstandard.decompress(f.read())
    return json.loads(data)


def normalize_name(name: str) -> str:
    """Lowercase and strip diacritics for fuzzy name matching."""
    decomposed = unicodedata.normalize("NFD", name)
    stripped = "".join(c for c in decomposed if unicodedata.category(c) != "Mn")
    return stripped.lower().strip()


def build_card_index(cards: list[dict]) -> dict[str, list[dict]]:
    """Index cards by normalized name. Each value is a list of printings."""
    index: dict[str, list[dict]] = defaultdict(list)
    for card in cards:
        key = normalize_name(card["CardName"])
        index[key].append(card)
    return index


def pick_printing(printings: list[dict]) -> dict:
    """Choose a single printing from the list. Prefer the earliest set
    (lexicographic on SetID is a rough proxy; good enough for old-school sets)."""
    return sorted(
        printings,
        key=lambda c: (c.get("SetID") or "", c.get("CollectorNo") or ""),
    )[0]


def format_dck_line(
    count: str, printing: dict | None, name: str, sideboard: bool
) -> str:
    prefix = "SB: " if sideboard else ""
    if printing is None:
        return f"{prefix}{count} [???:???] {name}"
    set_code = (printing.get("SetID") or "").upper()
    collector = printing.get("CollectorNo") or ""
    canonical = printing.get("CardName") or name
    return f"{prefix}{count} [{set_code}:{collector}] {canonical}"


def convert_deck(
    toml_path: Path,
    out_dir: Path,
    index: dict[str, list[dict]],
    missing: dict[str, set[str]],
) -> Path:
    with open(toml_path, "rb") as f:
        data = tomllib.load(f)

    deck_name = data.get("name") or toml_path.stem
    main_cards = data.get("main_cards") or []
    sideboard_cards = data.get("sideboard_cards") or []

    lines: list[str] = [f"NAME:{deck_name}"]

    for entries, is_sb in ((main_cards, False), (sideboard_cards, True)):
        for entry in entries:
            count, name = entry[0], entry[1]
            printings = index.get(normalize_name(name))
            if not printings:
                missing[toml_path.name].add(name)
                lines.append(format_dck_line(count, None, name, is_sb))
                continue
            lines.append(format_dck_line(count, pick_printing(printings), name, is_sb))

    out_path = out_dir / f"{toml_path.stem}.dck"
    out_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return out_path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    repo_root = Path(__file__).resolve().parent.parent
    parser.add_argument(
        "--rogues-dir",
        type=Path,
        default=repo_root / "assets" / "configs" / "rogues",
        help="Directory containing rogue .toml files",
    )
    parser.add_argument(
        "--out-dir",
        type=Path,
        default=repo_root / "decks",
        help="Output directory for .dck files",
    )
    parser.add_argument(
        "--cards-json",
        type=Path,
        default=repo_root / "assets" / "card_info" / "scryfall_cards.json.zst",
        help="Path to scryfall_cards.json.zst",
    )
    args = parser.parse_args()

    if not args.rogues_dir.is_dir():
        print(f"rogues dir not found: {args.rogues_dir}", file=sys.stderr)
        return 1

    args.out_dir = args.out_dir.resolve()
    args.out_dir.mkdir(parents=True, exist_ok=True)

    cards = load_scryfall_cards(args.cards_json)
    index = build_card_index(cards)

    toml_files = sorted(args.rogues_dir.glob("*.toml"))
    missing: dict[str, set[str]] = defaultdict(set)

    for toml_path in toml_files:
        out_path = convert_deck(toml_path, args.out_dir, index, missing)
        try:
            print(f"wrote {out_path.relative_to(repo_root)}")
        except ValueError:
            print(f"wrote {out_path}")

    if missing:
        total_unique = len({c for cs in missing.values() for c in cs})
        print(
            f"\nWarning: {total_unique} unique card name(s) not found in scryfall data "
            f"across {len(missing)} deck(s). Marked with [???:???] in output.",
            file=sys.stderr,
        )
        for deck, names in sorted(missing.items()):
            print(f"  {deck}:", file=sys.stderr)
            for name in sorted(names):
                print(f"    - {name}", file=sys.stderr)

    return 0


if __name__ == "__main__":
    sys.exit(main())
