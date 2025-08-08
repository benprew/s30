#!/usr/bin/env python3
"""
Script to clean up card images by removing any files not found in scryfall_cards.json.
This helps maintain consistency between the JSON data and stored images.
"""

import json
import re
import sys
from pathlib import Path


def sanitize_filename(name):
    """Convert card name to a safe filename format (same as download script)."""
    # Convert to lowercase and replace spaces/special chars with hyphens
    name = re.sub(r"[^\w\s-]", "", name.lower())
    name = re.sub(r"[-\s]+", "-", name)
    return name.strip("-")


def get_expected_filenames(cards_data):
    """Generate set of expected filenames from JSON data."""
    expected_files = set()

    for card in cards_data:
        # Try both processed format (CardName, SetID, CollectorNo) and raw format (name, set, collector_number)
        card_name = card.get("CardName") or card.get("name", "")
        set_code = card.get("SetID") or card.get("set", "")
        collector_number = card.get("CollectorNo") or card.get("collector_number", "")

        if not all([card_name, set_code, collector_number]):
            continue

        # Generate the expected filename (matches download script format)
        sanitized_name = sanitize_filename(card_name)
        expected_filename = f"{set_code}-{collector_number}-200-{sanitized_name}.png"
        expected_files.add(expected_filename)

    return expected_files


def main():
    if len(sys.argv) != 2:
        print("Usage: python cleanup_card_images.py <scryfall_cards.json>")
        sys.exit(1)

    json_file = sys.argv[1]

    # Image directory
    image_dir = Path("assets/art/carddata")

    if not image_dir.exists():
        print(f"Image directory {image_dir} does not exist")
        sys.exit(1)

    # Load JSON data
    try:
        with open(json_file, "r", encoding="utf-8") as f:
            cards_data = json.load(f)
    except Exception as e:
        print(f"Error reading JSON file: {e}")
        sys.exit(1)

    if not isinstance(cards_data, list):
        print("Error: JSON file should contain an array of card objects")
        sys.exit(1)

    print(f"Loaded {len(cards_data)} cards from JSON")

    # Get expected filenames
    expected_files = get_expected_filenames(cards_data)
    print(f"Generated {len(expected_files)} expected filenames")

    # Find all PNG files in the image directory
    existing_files = set()
    for png_file in image_dir.glob("*.png"):
        existing_files.add(png_file.name)

    print(f"Found {len(existing_files)} existing PNG files")

    # Find files to delete (exist but not expected)
    files_to_delete = existing_files - expected_files

    if not files_to_delete:
        print("No orphaned files found - all images match JSON data")
        return

    print(f"\nFound {len(files_to_delete)} orphaned files to delete:")

    # Sort for consistent output
    sorted_files = sorted(files_to_delete)

    # Show first 10 files as preview
    preview_count = min(10, len(sorted_files))
    for i in range(preview_count):
        print(f"  {sorted_files[i]}")

    if len(sorted_files) > preview_count:
        print(f"  ... and {len(sorted_files) - preview_count} more")

    # Confirm deletion
    response = input(f"\nDelete these {len(files_to_delete)} files? (y/N): ")
    if response.lower() != "y":
        print("Cancelled")
        return

    # Delete the files
    deleted_count = 0
    for filename in files_to_delete:
        file_path = image_dir / filename
        try:
            file_path.unlink()
            deleted_count += 1
        except Exception as e:
            print(f"Error deleting {filename}: {e}")

    print(f"\nDeleted {deleted_count}/{len(files_to_delete)} files")

    # Also report any expected files that are missing
    missing_files = expected_files - existing_files
    if missing_files:
        print(f"\nNote: {len(missing_files)} expected files are missing from disk")
        if len(missing_files) <= 5:
            for filename in sorted(missing_files):
                print(f"  Missing: {filename}")


if __name__ == "__main__":
    main()
