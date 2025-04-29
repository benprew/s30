#!/usr/bin/env python3

import argparse
import json
import sys
from PIL import Image

def find_subimages(image_path):
    """
    Finds rectangular subimages in a sprite sheet separated by a specific color.

    Args:
        image_path (str): Path to the image file.

    Returns:
        str: A JSON string representing a list of subimage rectangles
             (x, y, width, height).
             Returns an empty JSON list "[]" if the image is empty or
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

                    # Find the width of the subimage
                    sub_width = 0
                    for i in range(start_x, width):
                        if (i, start_y) in visited or pixels[i, start_y] == separator_color:
                            break
                        sub_width += 1

                    # Find the height of the subimage
                    sub_height = 0
                    for j in range(start_y, height):
                        if (start_x, j) in visited or pixels[start_x, j] == separator_color:
                            break
                        sub_height += 1

                    # --- Verification Step (Optional but recommended) ---
                    # Ensure the entire detected area is non-separator color.
                    # This helps catch non-rectangular shapes if the logic above
                    # has edge cases (though it shouldn't for simple grids).
                    is_rectangular = True
                    for j_verify in range(start_y, start_y + sub_height):
                        for i_verify in range(start_x, start_x + sub_width):
                            if pixels[i_verify, j_verify] == separator_color:
                                is_rectangular = False
                                # If not rectangular, we might need a more complex flood-fill
                                # or contour detection. For now, we'll just skip this shape.
                                # Mark the starting pixel visited to avoid re-processing.
                                visited.add(coord)
                                break
                        if not is_rectangular:
                            break

                    if not is_rectangular:
                        # print(f"Skipping non-rectangular area starting at {coord}", file=sys.stderr)
                        continue # Skip this starting pixel and let outer loops continue

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
        "image_file",
        help="Path to the sprite sheet image file (e.g., PNG)."
    )
    args = parser.parse_args()

    json_output = find_subimages(args.image_file)
    print(json_output)

if __name__ == "__main__":
    main()
