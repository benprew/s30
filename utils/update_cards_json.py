#!/usr/bin/env python3
"""
Process and filter Scryfall bulk card data into s30 card database format.
Supports streaming JSON/JSONL (.gz, .zst, or plain) from local files or URLs.
Outputs both formatted JSON and zstandard-compressed JSON (.zst).
"""

import argparse
import gzip
import io
import json
import logging
import sys
import urllib.request
from collections.abc import Collection
from pathlib import Path
from typing import Any, Iterator, NamedTuple

import zstandard

logger = logging.getLogger(__name__)

DEFAULT_ALLOWED_SETS: list[str] = ["2ed", "arn", "past", "atq", "phpr", "4ed"]
DEFAULT_EXCLUDED_NAMES: set[str] = {"Chaos Orb", "Shahrazad", "Word of Command"}


class VersionExclusion(NamedTuple):
    name: str
    set_id: str | None = None
    collector_number: str | None = None


def parse_version_exclusion(spec: str) -> VersionExclusion:
    """
    Parse a version exclusion string.
    Supported formats:
      - 'CardName:SetID' (e.g. 'El-Hajjâj:4ed')
      - 'CardName:SetID:CollectorNo' (e.g. 'Drudge Skeletons:4ed:107†')
      - 'CardName::CollectorNo' (e.g. 'El-Hajjâj::134†')
    """
    parts = spec.split(":")
    if len(parts) == 1:
        return VersionExclusion(name=parts[0].strip())
    if len(parts) == 2:
        return VersionExclusion(
            name=parts[0].strip(),
            set_id=parts[1].strip().lower() if parts[1].strip() else None,
        )
    return VersionExclusion(
        name=parts[0].strip(),
        set_id=parts[1].strip().lower() if parts[1].strip() else None,
        collector_number=parts[2].strip() if parts[2].strip() else None,
    )


def transform_raw_card(raw: dict[str, Any]) -> dict[str, Any]:
    """Transform a Scryfall card object into the s30 CardJSON schema."""
    prices = raw.get("prices") or {}
    price_usd = prices.get("usd")
    if not price_usd:
        price_usd = prices.get("eur")

    legalities = raw.get("legalities") or {}
    vintage_restricted = legalities.get("vintage") == "restricted"

    image_uris = raw.get("image_uris") or {}

    return {
        "CardName": raw.get("name"),
        "ManaCost": raw.get("mana_cost"),
        "Colors": raw.get("colors") or [],
        "ColorIdentity": raw.get("color_identity") or [],
        "Keywords": raw.get("keywords") or [],
        "TypeLine": raw.get("type_line"),
        "Text": raw.get("oracle_text"),
        "Power": raw.get("power"),
        "Toughness": raw.get("toughness"),
        "SetName": raw.get("set_name"),
        "SetID": raw.get("set"),
        "CollectorNo": raw.get("collector_number"),
        "Rarity": raw.get("rarity"),
        "Frame": raw.get("frame"),
        "FlavorText": raw.get("flavor_text"),
        "FrameEffects": raw.get("frame_effects"),
        "Watermark": raw.get("watermark"),
        "Artist": raw.get("artist"),
        "ManaProduction": raw.get("produced_mana"),
        "PriceUSD": price_usd,
        "VintageRestricted": vintage_restricted,
        "PngURL": image_uris.get("png"),
        "ArtURL": image_uris.get("art_crop"),
        "BorderCropURL": image_uris.get("border_crop"),
    }


def iter_json_records(
    stream: io.IOBase | gzip.GzipFile,
) -> Iterator[dict[str, Any]]:
    """Yield JSON objects from a stream, handling both JSON arrays and JSONL."""
    peek_bytes = stream.read(1024)
    if not peek_bytes:
        return

    # Check if stream begins with a JSON array '['
    first_char = peek_bytes.lstrip()[:1]
    if first_char == b"[":
        # Full JSON array
        full_content = peek_bytes + stream.read()
        records = json.loads(full_content.decode("utf-8"))
        if isinstance(records, list):
            yield from records
        elif isinstance(records, dict):
            yield records
        return

    # Treat as JSON Lines
    buffer = io.BytesIO(peek_bytes + stream.read())
    text_stream = io.TextIOWrapper(buffer, encoding="utf-8")
    for line in text_stream:
        line = line.strip()
        if not line:
            continue
        try:
            yield json.loads(line)
        except json.JSONDecodeError as err:
            logger.warning(f"Skipping invalid JSON line: {err}")


def open_input_stream(source: str) -> io.IOBase | gzip.GzipFile:
    """Open input stream from local file or HTTP URL with decompression."""
    raw_stream: io.IOBase
    if source.startswith("http://") or source.startswith("https://"):
        req = urllib.request.Request(
            source, headers={"User-Agent": "s30-card-updater/1.0"}
        )
        raw_stream = urllib.request.urlopen(req)
    else:
        raw_stream = open(source, "rb")

    source_lower = source.lower()
    if source_lower.endswith(".gz"):
        return gzip.GzipFile(fileobj=raw_stream)
    if source_lower.endswith(".zst"):
        dctx = zstandard.ZstdDecompressor()
        decompressed_data = dctx.decompress(raw_stream.read())
        return io.BytesIO(decompressed_data)

    return raw_stream


def process_card_records(
    records: Iterator[dict[str, Any]],
    allowed_sets: list[str],
    excluded_names: set[str],
) -> Collection[dict[str, Any]]:
    """Filter and transform card records."""
    print(f"allowed_set: {allowed_sets}")
    allowed_sets_lower = [s.lower() for s in allowed_sets]
    excluded_names_lower = {n.lower() for n in excluded_names}

    results: dict[str, dict[str, Any]] = {}

    for raw in records:
        # Detect if record is already transformed or raw Scryfall
        is_already_transformed = "CardName" in raw and "SetID" in raw

        if is_already_transformed:
            name = str(raw.get("CardName", ""))
            set_id = str(raw.get("SetID", ""))
            transformed = raw
        else:
            name = str(raw.get("name", ""))
            set_id = str(raw.get("set", ""))
            transformed = transform_raw_card(raw)

        if set_id.lower() not in allowed_sets_lower:
            continue

        if name.lower() in excluded_names_lower:
            continue

        if name in results and allowed_sets_lower.index(
            set_id
        ) > allowed_sets_lower.index(results[name]["SetID"]):
            print(
                f"skipping {name} to results (for {set_id}), "
                f"new:{allowed_sets_lower.index(set_id)}, "
                f"curr:{allowed_sets_lower.index(results[name]['SetID'])}"
            )
            continue

        print(
            f"adding {name} to results (for {set_id}), new:{allowed_sets_lower.index(set_id)}"
        )
        results[name] = transformed

    return results.values()


def save_json_and_zst(
    cards: Collection[dict[str, Any]],
    output_json_path: Path,
    output_zst_path: Path | None = None,
) -> None:
    """Save formatted JSON and compressed zstd file."""
    output_json_path.parent.mkdir(parents=True, exist_ok=True)
    json_bytes = json.dumps(list(cards), indent=2, ensure_ascii=False).encode("utf-8")

    with open(output_json_path, "wb") as f:
        f.write(json_bytes)
    logger.info(f"Saved {len(cards)} cards to {output_json_path}")

    if output_zst_path is not None:
        output_zst_path.parent.mkdir(parents=True, exist_ok=True)
        cctx = zstandard.ZstdCompressor(level=19)
        compressed = cctx.compress(json_bytes)
        with open(output_zst_path, "wb") as f:
            f.write(compressed)
        logger.info(f"Saved compressed card database to {output_zst_path}")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Filter and format Scryfall card data for s30."
    )
    parser.add_argument(
        "input_source",
        help="Path or URL to JSON/JSONL card data (.json, .jsonl, .gz, .zst)",
    )
    parser.add_argument(
        "--sets",
        default=",".join(DEFAULT_ALLOWED_SETS),
        help="Comma-separated list of allowed set codes (default: %(default)s), overlapping cards prefer the set given first",
    )
    parser.add_argument(
        "--exclude-name",
        action="append",
        default=[],
        help="Card name to exclude completely (can be specified multiple times)",
    )
    parser.add_argument(
        "--exclude-version",
        action="append",
        default=[],
        help="Card version to exclude in format Name:SetID or Name:SetID:CollectorNo",
    )
    parser.add_argument(
        "--output-json",
        default="assets/card_info/scryfall_cards.json",
        help="Output path for scryfall_cards.json (default: %(default)s)",
    )
    parser.add_argument(
        "--output-zst",
        default="assets/card_info/scryfall_cards.json.zst",
        help="Output path for scryfall_cards.json.zst (default: %(default)s)",
    )
    parser.add_argument(
        "--skip-zst",
        action="store_true",
        help="Skip writing the .zst compressed file",
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="Enable verbose logging",
    )

    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO if args.verbose else logging.WARNING,
        format="%(levelname)s: %(message)s",
    )

    allowed_sets = [s.strip().lower() for s in args.sets.split(",") if s.strip()]
    excluded_names = set(DEFAULT_EXCLUDED_NAMES)
    for name in args.exclude_name:
        excluded_names.add(name.strip())

    print(f"Reading input from: {args.input_source}")
    try:
        stream = open_input_stream(args.input_source)
    except Exception as e:
        logger.error(f"Failed to open input {args.input_source}: {e}")
        sys.exit(1)

    with stream:
        records = iter_json_records(stream)
        cards = process_card_records(
            records=records,
            allowed_sets=allowed_sets,
            excluded_names=excluded_names,
        )

    print(f"Processed {len(cards)} matching cards")

    output_json = Path(args.output_json)
    output_zst = Path(args.output_zst) if not args.skip_zst else None

    save_json_and_zst(cards, output_json, output_zst)

    print(f"Successfully updated {output_json}")
    if output_zst:
        print(f"Successfully updated {output_zst}")


if __name__ == "__main__":
    main()
