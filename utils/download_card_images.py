#!/usr/bin/env python3
"""
Script to download and process Magic: The Gathering card images from Scryfall JSON data.
Downloads PNG images, resizes them to 200px width, and compresses them with pngquant.
"""

import json
import os
import re
import subprocess
import sys
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path


def sanitize_filename(name):
    """Convert card name to a safe filename format."""
    # Convert to lowercase and replace spaces/special chars with hyphens
    name = re.sub(r"[^\w\s-]", "", name.lower())
    name = re.sub(r"[-\s]+", "-", name)
    return name.strip("-")


def download_and_process_card(card_data, output_dir):
    """Download and process a single card image."""
    try:
        # Extract card information
        card_name = card_data.get("name", "")
        set_code = card_data.get("set", "")
        collector_number = card_data.get("collector_number", "")

        # Get PNG image URL
        image_uris = card_data.get("image_uris", {})
        png_url = image_uris.get("png")

        if not png_url:
            print(f"No PNG URL found for {card_name}")
            return False

        # Create sanitized filename
        sanitized_name = sanitize_filename(card_name)
        base_filename = f"{set_code}-{collector_number}-{sanitized_name}"
        original_filename = f"{base_filename}.png"
        resized_filename = f"{set_code}-{collector_number}-200-{sanitized_name}.png"

        original_path = output_dir / original_filename
        resized_path = output_dir / resized_filename

        # Check if the final resized image already exists
        if resized_path.exists():
            print(
                f"Skipping {card_name} ({set_code}-{collector_number}) - already exists"
            )
            return True

        print(f"Processing {card_name} ({set_code}-{collector_number})...")

        # Download original image
        print(f"  Downloading from {png_url}")
        urllib.request.urlretrieve(png_url, original_path)

        resize_size = "300x"
        # Resize image using ImageMagick convert
        print(f"  Resizing to {resize_size} width...")
        convert_cmd = [
            "convert",
            str(original_path),
            "-resize",
            resize_size,
            str(resized_path),
        ]

        result = subprocess.run(convert_cmd, capture_output=True, text=True)
        if result.returncode != 0:
            print(f"  Error resizing image: {result.stderr}")
            return False

        # Run pngquant for compression
        print("  Compressing with pngquant...")
        pngquant_cmd = [
            "pngquant",
            "--quality=60-80",
            "--force",
            "--output",
            str(resized_path),
            str(resized_path),
        ]

        result = subprocess.run(pngquant_cmd, capture_output=True, text=True)
        if result.returncode != 0:
            print(f"  Warning: pngquant failed: {result.stderr}")
            # Continue anyway, the resized image is still usable

        # Clean up original downloaded file
        original_path.unlink()

        print(f"  Saved as {resized_filename}")
        return True

    except Exception as e:
        print(f"Error processing {card_data}: {e}")
        return False


def main():
    if len(sys.argv) != 2:
        print("Usage: python download_card_images.py <json_file>")
        sys.exit(1)

    json_file = sys.argv[1]

    # Create output directory
    output_dir = Path("assets/art/carddata")
    output_dir.mkdir(parents=True, exist_ok=True)

    # Check for required tools
    try:
        subprocess.run(["convert", "--version"], capture_output=True, check=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        print(
            "Error: ImageMagick 'convert' command not found. Please install ImageMagick."
        )
        sys.exit(1)

    try:
        subprocess.run(["pngquant", "--version"], capture_output=True, check=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        print("Warning: pngquant not found. Images will be resized but not compressed.")

    # Load and process JSON data
    try:
        with open(json_file, "r", encoding="utf-8") as f:
            cards_data = json.load(f)
    except Exception as e:
        print(f"Error reading JSON file: {e}")
        sys.exit(1)

    if not isinstance(cards_data, list):
        print("Error: JSON file should contain an array of card objects")
        sys.exit(1)

    print(f"Processing {len(cards_data)} cards...")

    success_count = 0
    max_workers = min(
        8, len(cards_data)
    )  # Limit concurrent downloads to avoid overwhelming the server

    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        # Submit all tasks
        future_to_card = {
            executor.submit(download_and_process_card, card, output_dir): i
            for i, card in enumerate(cards_data, 1)
        }

        # Process completed tasks
        for future in as_completed(future_to_card):
            card_num = future_to_card[future]
            print(f"\n[{card_num}/{len(cards_data)}]", end=" ")
            try:
                if future.result():
                    success_count += 1
            except Exception as e:
                print(f"Task failed with exception: {e}")

    print(
        f"\n\nCompleted: {success_count}/{len(cards_data)} cards processed successfully"
    )


if __name__ == "__main__":
    main()
