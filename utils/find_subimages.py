#!/usr/bin/env python3

import argparse
import json
import sys
from PIL import Image


def find_subimages(image_path, interactive=False):
    """
    Finds rectangular subimages in a sprite sheet separated by a specific color.

    Args:
        image_path (str): Path to the image file.
        interactive (bool): If True, prompts user to name each rectangle.

    Returns:
        str: A JSON string representing subimage rectangles.
             If interactive=False: list of rectangles (x, y, width, height).
             If interactive=True: dictionary mapping names to rectangles.
             Returns an empty JSON list/dict if the image is empty or
             cannot be opened.

    Raises:
        FileNotFoundError: If the image file does not exist.
        Exception: For other image processing errors.
    """
    try:
        with Image.open(image_path) as img:
            # Ensure image is in RGBA format for consistent color handling
            img = img.convert("RGBA")
            pixels = img.load()
            width, height = img.size

            if width == 0 or height == 0:
                return "[]"

            separator_color = pixels[0, 0]
            visited = set()
            subimage_infos = []

            for y in range(height):
                for x in range(width):
                    coord = (x, y)
                    # Skip visited pixels or separator pixels
                    if coord in visited or pixels[x, y] == separator_color:
                        continue

                    # Found the top-left corner of a new subimage
                    start_x, start_y = x, y

                    # Find the width of the subimage (scan until separator color)
                    sub_width = 0
                    for i in range(start_x, width):
                        if pixels[i, start_y] == separator_color:
                            break
                        sub_width += 1

                    # Find the height of the subimage (scan until separator color)
                    sub_height = 0
                    for j in range(start_y, height):
                        if pixels[start_x, j] == separator_color:
                            break
                        sub_height += 1

                    # Verify the entire area is non-separator and bounded by separator
                    is_valid = True

                    # Check interior has no separator pixels
                    for j_verify in range(start_y, start_y + sub_height):
                        for i_verify in range(start_x, start_x + sub_width):
                            if pixels[i_verify, j_verify] == separator_color:
                                is_valid = False
                                break
                        if not is_valid:
                            break

                    # Check all 4 edges are bounded by separator or image boundary
                    if is_valid:
                        # Top edge: row above must be separator or image boundary
                        if start_y > 0:
                            for i in range(start_x, start_x + sub_width):
                                if pixels[i, start_y - 1] != separator_color:
                                    is_valid = False
                                    break

                        # Bottom edge: row below must be separator or image boundary
                        if is_valid and start_y + sub_height < height:
                            for i in range(start_x, start_x + sub_width):
                                if pixels[i, start_y + sub_height] != separator_color:
                                    is_valid = False
                                    break

                        # Left edge: column to left must be separator or image boundary
                        if is_valid and start_x > 0:
                            for j in range(start_y, start_y + sub_height):
                                if pixels[start_x - 1, j] != separator_color:
                                    is_valid = False
                                    break

                        # Right edge: column to right must be separator or image boundary
                        if is_valid and start_x + sub_width < width:
                            for j in range(start_y, start_y + sub_height):
                                if pixels[start_x + sub_width, j] != separator_color:
                                    is_valid = False
                                    break

                    if not is_valid:
                        for j_mark in range(start_y, start_y + sub_height):
                            for i_mark in range(start_x, start_x + sub_width):
                                visited.add((i_mark, j_mark))
                        continue

                    # --- Store and Mark ---
                    info = {
                        "x": start_x,
                        "y": start_y,
                        "width": sub_width,
                        "height": sub_height,
                    }
                    subimage_infos.append(info)

                    # Mark the pixels within the found rectangle as visited
                    for j_mark in range(start_y, start_y + sub_height):
                        for i_mark in range(start_x, start_x + sub_width):
                            visited.add((i_mark, j_mark))

            if interactive:
                named_subimages = {}
                print(f"\nFound {len(subimage_infos)} rectangles. Please name each one:", file=sys.stderr)
                print("(Press Enter to skip or use auto-generated name)\n", file=sys.stderr)

                for i, info in enumerate(subimage_infos):
                    print(f"Rectangle {i}:", file=sys.stderr)
                    print(f"  Position: ({info['x']}, {info['y']})", file=sys.stderr)
                    print(f"  Size: {info['width']}x{info['height']}", file=sys.stderr)
                    print(f"  Name: ", end="", file=sys.stderr)
                    sys.stderr.flush()

                    name = input().strip()
                    if not name:
                        name = f"rect_{i}"

                    if name in named_subimages:
                        original_name = name
                        counter = 1
                        while name in named_subimages:
                            name = f"{original_name}_{counter}"
                            counter += 1
                        print(f"  (Name already used, using '{name}' instead)", file=sys.stderr)

                    named_subimages[name] = info
                    print(file=sys.stderr)

                return json.dumps(named_subimages, indent=2)
            else:
                return json.dumps(subimage_infos, indent=2)

    except FileNotFoundError:
        print(f"Error: File not found at {image_path}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error processing image {image_path}: {e}", file=sys.stderr)
        sys.exit(1)


def main():
    """Parses command line arguments and runs the subimage finding process."""
    parser = argparse.ArgumentParser(
        description="Find subimages in a sprite sheet separated by the color at pixel (0,0) and output their coordinates and dimensions as JSON."
    )
    parser.add_argument(
        "image_file", help="Path to the sprite sheet image file (e.g., PNG)."
    )
    parser.add_argument(
        "--interactive",
        "-i",
        action="store_true",
        help="Interactive mode: prompt for name of each rectangle. Output will be a JSON map instead of a list.",
    )
    parser.add_argument(
        "--output",
        "-o",
        type=str,
        help="Output filename for JSON. If not provided, prints to stdout.",
    )
    args = parser.parse_args()

    json_output = find_subimages(args.image_file, interactive=args.interactive)

    if args.output:
        with open(args.output, 'w') as f:
            f.write(json_output)
        print(f"Wrote output to {args.output}", file=sys.stderr)
    else:
        print(json_output)


if __name__ == "__main__":
    main()
