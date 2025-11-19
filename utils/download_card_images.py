#!/usr/bin/env python3
"""
Script to download and process Magic: The Gathering card images from Scryfall JSON data.
Downloads PNG images, resizes them to 200px width, and compresses them with pngquant.
Supports both regular JSON and zstandard-compressed JSON (.zst) files.
"""

import argparse
import json
import logging
import re
import shutil
import subprocess
import sys
import tempfile
import urllib.request
import zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from typing import Dict, Any, List

try:
    import zstandard as zstd

    HAS_ZSTD = True
except ImportError:
    HAS_ZSTD = False

logger = logging.getLogger(__name__)

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


def download_and_process_card(
    card_data: Dict[str, Any], zip_path: Path, output_dir: Path
) -> tuple[bool, str]:
    """Download and process a single card image, saving to output_dir."""
    try:
        card_name = card_data.get("CardName", "")
        set_code = card_data.get("SetID", "")
        collector_number = card_data.get("CollectorNo", "")
        png_url = card_data.get("PngURL")

        if not png_url:
            logger.error(f"No PNG URL found for {card_name}")
            return False, ""

        sanitized_name = sanitize_filename(card_name)
        resized_filename = f"{set_code}-{collector_number}-200-{sanitized_name}.png"

        if zip_path.exists():
            with zipfile.ZipFile(zip_path, "r") as zf:
                if resized_filename in zf.namelist():
                    logger.info(
                        f"Skipping {card_name} ({set_code}-{collector_number}) - already exists in zip"
                    )
                    return True, ""

        logger.info(f"Processing {card_name} ({set_code}-{collector_number})")

        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = Path(temp_dir)
            original_path = temp_path / "original.png"
            resized_path = temp_path / "resized.png"

            logger.info(f"Downloading from {png_url}")
            urllib.request.urlretrieve(png_url, original_path)

            resize_size = "300x"
            logger.info(f"Resizing to {resize_size} width")
            convert_cmd = [
                "convert",
                str(original_path),
                "-resize",
                resize_size,
                str(resized_path),
            ]

            result = subprocess.run(convert_cmd, capture_output=True, text=True)
            if result.returncode != 0:
                logger.error(f"Error resizing image: {result.stderr}")
                return False, ""

            logger.info("Compressing with pngquant")
            pngquant_cmd = [
                "pngquant",
                "--quality=60-80",
                "--force",
                "--strip",
                "--output",
                str(resized_path),
                str(resized_path),
            ]

            result = subprocess.run(pngquant_cmd, capture_output=True, text=True)
            if result.returncode != 0:
                logger.warning(f"pngquant failed: {result.stderr}")

            output_path = output_dir / resized_filename
            shutil.copy(resized_path, output_path)

        print(f"Completed: {card_name} ({set_code}-{collector_number})")
        return True, resized_filename

    except Exception as e:
        logger.error(f"Error processing {card_name}: {e}")
        return False, ""


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Download and process Magic: The Gathering card images from Scryfall JSON data."
    )
    parser.add_argument("json_file", help="Path to JSON or JSON.zst file containing card data")
    parser.add_argument(
        "-v", "--verbose",
        action="store_true",
        help="Enable verbose output (shows detailed progress)"
    )
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO if args.verbose else logging.WARNING,
        format="%(message)s"
    )

    zip_path = Path("assets/art/cardimages.zip")
    zip_path.parent.mkdir(parents=True, exist_ok=True)

    try:
        subprocess.run(["convert", "--version"], capture_output=True, check=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        logger.error("ImageMagick 'convert' command not found. Please install ImageMagick.")
        sys.exit(1)

    try:
        subprocess.run(["pngquant", "--version"], capture_output=True, check=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        logger.warning("pngquant not found. Images will be resized but not compressed.")

    try:
        cards_data = load_json_file(args.json_file)
    except Exception as e:
        logger.error(f"Error reading JSON file: {e}")
        sys.exit(1)

    if not isinstance(cards_data, list):
        logger.error("JSON file should contain an array of card objects")
        sys.exit(1)

    logger.info(f"Processing {len(cards_data)} cards...")

    success_count: int = 0
    max_workers: int = min(8, len(cards_data))
    processed_files: List[str] = []

    with tempfile.TemporaryDirectory() as temp_output_dir:
        output_dir = Path(temp_output_dir)

        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_card = {
                executor.submit(download_and_process_card, card, zip_path, output_dir): i
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


if __name__ == "__main__":
    main()
