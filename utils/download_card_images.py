#!/usr/bin/env python3
"""
Script to download and process Magic: The Gathering card images from Scryfall JSON data.
Downloads, resizes and optimizes PNG images.
Supports both regular JSON and zstandard-compressed JSON (.zst) files.
"""

import argparse
import io
import json
import logging
import re
import sys
import tempfile
import urllib.request
import zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from typing import Any, Dict, List, Set

from PIL import Image

try:
    import zstandard as zstd

    HAS_ZSTD = True
except ImportError:
    HAS_ZSTD = False

logger = logging.getLogger(__name__)

CARD_WIDTH = 245

# TODO: the transform cards have 2 images in for a single card.
# Probably save the first image as the untransformed name?
# And then split the cardname in domain.Card
# see pyretic-prankster // glistening-goremonger
# Or... just ignore them for now


def load_json_file(file_path: str) -> List[Dict[str, Any]]:
    """Load JSON from either compressed (.zst) or uncompressed file."""
    path = Path(file_path)

    if path.suffix == ".zst":
        if not HAS_ZSTD:
            logger.error(
                "zstandard module not installed. Install with: pip install zstandard"
            )
            sys.exit(1)

        logger.info(f"Loading compressed JSON from {file_path}")
        with open(file_path, "rb") as f:
            dctx = zstd.ZstdDecompressor()
            decompressed = dctx.decompress(f.read())
            return json.loads(decompressed.decode("utf-8"))
    else:
        logger.info(f"Loading JSON from {file_path}")
        with open(file_path, "r", encoding="utf-8") as f:
            return json.load(f)


def sanitize_filename(name: str) -> str:
    """Convert card name to a safe filename format."""
    name = re.sub(r"[^\w\s-]", "", name.lower())
    name = re.sub(r"[-\s]+", "-", name)
    return name.strip("-")


def image_filename(card_data: Dict[str, Any]) -> str:
    """Return the archive filename used for a card image."""
    card_name = card_data.get("CardName", "")
    set_code = card_data.get("SetID", "")
    collector_number = card_data.get("CollectorNo", "")
    sanitized_name = sanitize_filename(card_name)
    return f"{set_code}-{collector_number}-200-{sanitized_name}.png"


def download_and_process_card(
    card_data: Dict[str, Any], existing_files: Set[str], output_dir: Path
) -> tuple[bool, str]:
    """Download and process a single card image, saving to output_dir."""
    card_name = card_data.get("CardName", "")
    try:
        set_code = card_data.get("SetID", "")
        collector_number = card_data.get("CollectorNo", "")
        png_url = card_data.get("PngURL")

        if not png_url:
            logger.error(f"No PNG URL found for {card_name}")
            return False, ""

        resized_filename = image_filename(card_data)

        if resized_filename in existing_files:
            logger.info(
                f"Skipping {card_name} ({set_code}-{collector_number}) "
                "- already exists in zip"
            )
            return True, ""

        logger.info(f"Processing {card_name} ({set_code}-{collector_number})")

        logger.info(f"Downloading from {png_url}")
        with urllib.request.urlopen(png_url, timeout=30) as response:
            image_data = response.read()

        output_path = output_dir / resized_filename
        with Image.open(io.BytesIO(image_data)) as original:
            target_height = round(original.height * CARD_WIDTH / original.width)
            logger.info(f"Resizing to {CARD_WIDTH}x{target_height}")
            resized = original.resize(
                (CARD_WIDTH, target_height), Image.Resampling.LANCZOS
            )
            if resized.mode == "RGBA":
                resized = resized.quantize(colors=256, method=Image.Quantize.FASTOCTREE)
            else:
                resized = resized.convert("RGB").quantize(
                    colors=256, method=Image.Quantize.MEDIANCUT
                )
            resized.save(output_path, format="PNG", optimize=True)

        print(f"Completed: {card_name} ({set_code}-{collector_number})")
        return True, resized_filename

    except Exception as e:
        logger.error(f"Error processing {card_name}: {e}")
        return False, ""


def main() -> None:
    parser = argparse.ArgumentParser(
        description=(
            "Download and process Magic: The Gathering card images "
            "from Scryfall JSON data."
        )
    )
    parser.add_argument(
        "json_file", help="Path to JSON or JSON.zst file containing card data"
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="Enable verbose output (shows detailed progress)",
    )
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO if args.verbose else logging.WARNING, format="%(message)s"
    )

    zip_path = Path("assets/art/cardimages.zip")
    zip_path.parent.mkdir(parents=True, exist_ok=True)

    try:
        cards_data = load_json_file(args.json_file)
    except Exception as e:
        logger.error(f"Error reading JSON file: {e}")
        sys.exit(1)

    if not isinstance(cards_data, list):
        logger.error("JSON file should contain an array of card objects")
        sys.exit(1)

    if not cards_data:
        logger.error("Card data is empty")
        sys.exit(1)

    logger.info(f"Processing {len(cards_data)} cards...")

    success_count: int = 0
    max_workers: int = min(8, len(cards_data))
    processed_files: List[str] = []

    existing_files: Set[str] = set()
    if zip_path.exists():
        try:
            with zipfile.ZipFile(zip_path, "r") as zf:
                existing_files = set(zf.namelist())
        except zipfile.BadZipFile as e:
            logger.error(f"Invalid existing ZIP archive: {e}")
            sys.exit(1)

    expected_files = {image_filename(card) for card in cards_data}

    with tempfile.TemporaryDirectory() as temp_output_dir:
        output_dir = Path(temp_output_dir)

        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_card = {
                executor.submit(
                    download_and_process_card, card, existing_files, output_dir
                ): i
                for i, card in enumerate(cards_data, 1)
            }

            for future in as_completed(future_to_card):
                try:
                    success, filename = future.result()
                    if success:
                        success_count += 1
                        if filename:
                            processed_files.append(filename)
                except Exception as e:
                    logger.error(f"Task failed with exception: {e}")

        if processed_files:
            logger.info(f"Adding {len(processed_files)} files to zip archive...")
            with zipfile.ZipFile(zip_path, "a") as zf:
                for filename in processed_files:
                    file_path = output_dir / filename
                    zf.write(file_path, filename)
                    logger.info(f"Added {filename}")

    print(f"Completed: {success_count}/{len(cards_data)} cards processed successfully")

    available_files = existing_files | set(processed_files)
    missing_files = expected_files - available_files
    if success_count != len(cards_data) or missing_files:
        logger.error(f"Failed to produce {len(missing_files)} card images")
        sys.exit(1)


if __name__ == "__main__":
    main()
