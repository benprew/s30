#!/usr/bin/env python3
"""
Visualize all connections for each sprite in a single image.
Shows the central sprite with all its connected neighbors arranged around it.
"""

import argparse
from PIL import Image, ImageDraw, ImageFont
import numpy as np
from pathlib import Path
from analyze_edges import (
    extract_sprites,
    extract_all_edges,
    analyze_connections,
    CORNER_POSITIONS,
    COMPATIBLE_EDGES,
)


def get_sprite_offset_for_connection(from_edge, to_edge):
    """
    Calculate the offset needed to position sprite B relative to sprite A
    so that their connecting edges align.

    For edges to connect, the "from" corners should be at the same point
    (the edges diverge from a shared vertex).
    """
    from_corner_a, to_corner_a = from_edge
    from_corner_b, to_corner_b = to_edge

    # Get positions for the "from" corners (where edges start)
    x1_a, y1_a = CORNER_POSITIONS[from_corner_a]
    x1_b, y1_b = CORNER_POSITIONS[from_corner_b]

    # Calculate offset to align B's from_corner with A's from_corner
    offset_x = x1_a - x1_b
    offset_y = y1_a - y1_b

    return (offset_x, offset_y)


def create_sprite_connections_visualization(
    sprite_a, sprites_grid, sprite_id_a, connections, all_edges, max_per_edge=3
):
    """
    Create visualization showing a sprite with all its connections as separate pairs.
    Each pair is drawn independently in a grid layout.
    """
    edges_a = all_edges[sprite_id_a]

    if len(edges_a) == 0:
        return None

    # Collect all connection pairs
    connection_pairs = []

    for direction_a, edge_a in edges_a.items():
        if direction_a not in connections[sprite_id_a]:
            continue

        edge_length, matches = connections[sprite_id_a][direction_a]

        if len(matches) == 0:
            continue

        # Take top N matches per edge
        for i, (sprite_id_b, direction_b, length_b, similarity) in enumerate(
            matches[:max_per_edge]
        ):
            row_b, col_b = sprite_id_b
            sprite_b = sprites_grid[row_b][col_b]

            from_a, to_a = map(int, direction_a.split("->"))
            from_b, to_b = map(int, direction_b.split("->"))

            offset_x, offset_y = get_sprite_offset_for_connection(
                (from_a, to_a), (from_b, to_b)
            )

            connection_pairs.append(
                {
                    "sprite": sprite_b,
                    "sprite_id": sprite_id_b,
                    "offset": (offset_x, offset_y),
                    "edge_a": direction_a,
                    "edge_b": direction_b,
                    "from_a": from_a,
                    "to_a": to_a,
                    "from_b": from_b,
                    "to_b": to_b,
                    "similarity": similarity,
                }
            )

    if len(connection_pairs) == 0:
        return None

    sprite_w = sprite_a.width
    sprite_h = sprite_a.height

    # Calculate size needed for each pair
    pair_spacing = 20
    pair_width = sprite_w * 3 + pair_spacing  # Space for 2 sprites with gap
    pair_height = sprite_h + 40  # Space for sprites + label

    # Arrange pairs in a grid
    pairs_per_row = 2
    num_rows = (len(connection_pairs) + pairs_per_row - 1) // pairs_per_row

    margin = 50
    title_height = 30
    canvas_width = pairs_per_row * pair_width + margin * 2
    canvas_height = num_rows * pair_height + margin * 2 + title_height

    canvas = Image.new("RGBA", (canvas_width, canvas_height), (240, 240, 240, 255))
    draw = ImageDraw.Draw(canvas)

    # Draw title
    title = f"Sprite ({sprite_id_a[0]},{sprite_id_a[1]}) - All Connections"
    draw.text((10, 10), title, fill=(0, 0, 0))

    corner_colors = {0: (255, 0, 0), 1: (0, 255, 0), 2: (0, 0, 255), 3: (255, 255, 0)}

    # Draw each pair separately
    for idx, conn in enumerate(connection_pairs):
        row = idx // pairs_per_row
        col = idx % pairs_per_row

        # Base position for this pair
        base_x = margin + col * pair_width
        base_y = margin + title_height + row * pair_height

        # Calculate positions for sprite A and sprite B in this pair
        offset_x, offset_y = conn["offset"]

        # Determine bounds for this pair
        min_x = min(0, offset_x)
        max_x = max(sprite_w, offset_x + sprite_w)
        min_y = min(0, offset_y)
        max_y = max(sprite_h, offset_y + sprite_h)

        # Center the pair in its allocated space
        pair_content_width = max_x - min_x
        pair_content_height = max_y - min_y
        center_offset_x = (pair_width - pair_content_width) // 2
        center_offset_y = (pair_height - pair_content_height - 20) // 2

        sprite_a_x = base_x + center_offset_x - min_x
        sprite_a_y = base_y + center_offset_y - min_y

        sprite_b_x = sprite_a_x + offset_x
        sprite_b_y = sprite_a_y + offset_y

        # Draw sprite B
        sprite_b_rgba = conn["sprite"].convert("RGBA")
        draw.rectangle(
            [
                sprite_b_x - 1,
                sprite_b_y - 1,
                sprite_b_x + sprite_w + 1,
                sprite_b_y + sprite_h + 1,
            ],
            outline=(100, 100, 200),
            width=1,
        )
        canvas.paste(sprite_b_rgba, (sprite_b_x, sprite_b_y), sprite_b_rgba)

        # Draw edge line on sprite B
        x1_b = sprite_b_x + CORNER_POSITIONS[conn["from_b"]][0]
        y1_b = sprite_b_y + CORNER_POSITIONS[conn["from_b"]][1]
        x2_b = sprite_b_x + CORNER_POSITIONS[conn["to_b"]][0]
        y2_b = sprite_b_y + CORNER_POSITIONS[conn["to_b"]][1]
        draw.line([x1_b, y1_b, x2_b, y2_b], fill=(0, 0, 255), width=2)

        # Draw sprite A
        sprite_a_rgba = sprite_a.convert("RGBA")
        draw.rectangle(
            [
                sprite_a_x - 2,
                sprite_a_y - 2,
                sprite_a_x + sprite_w + 2,
                sprite_a_y + sprite_h + 2,
            ],
            outline=(200, 0, 0),
            width=2,
        )
        canvas.paste(sprite_a_rgba, (sprite_a_x, sprite_a_y), sprite_a_rgba)

        # Draw edge line on sprite A
        x1_a = sprite_a_x + CORNER_POSITIONS[conn["from_a"]][0]
        y1_a = sprite_a_y + CORNER_POSITIONS[conn["from_a"]][1]
        x2_a = sprite_a_x + CORNER_POSITIONS[conn["to_a"]][0]
        y2_a = sprite_a_y + CORNER_POSITIONS[conn["to_a"]][1]
        draw.line([x1_a, y1_a, x2_a, y2_a], fill=(255, 0, 0), width=2)

        # Draw corner markers on sprite A
        for corner_id, (x, y) in CORNER_POSITIONS.items():
            draw.ellipse(
                [
                    sprite_a_x + x - 2,
                    sprite_a_y + y - 2,
                    sprite_a_x + x + 2,
                    sprite_a_y + y + 2,
                ],
                fill=corner_colors[corner_id],
                outline=(0, 0, 0),
            )

        # Label sprites
        draw.text(
            (sprite_a_x + 3, sprite_a_y + 3),
            f"({sprite_id_a[0]},{sprite_id_a[1]})",
            fill=(200, 0, 0),
        )
        sprite_id_b = conn["sprite_id"]
        draw.text(
            (sprite_b_x + 3, sprite_b_y + 3),
            f"({sprite_id_b[0]},{sprite_id_b[1]})",
            fill=(0, 0, 200),
        )

        # Label below the pair
        label_y = base_y + pair_height - 15
        label_text = f"{conn['edge_a']} ↔ ({sprite_id_b[0]},{sprite_id_b[1]}) [{conn['edge_b']}] {conn['similarity']:.1f}%"
        draw.text((base_x + 5, label_y), label_text, fill=(60, 60, 60))

    return canvas


def main():
    parser = argparse.ArgumentParser(description="Visualize all connections per sprite")
    parser.add_argument(
        "--output-dir",
        default="sprite_all_connections",
        help="Directory to save visualizations",
    )
    parser.add_argument(
        "--max-per-edge",
        type=int,
        default=3,
        help="Maximum connections to show per edge",
    )
    args = parser.parse_args()

    # Create output directory
    output_dir = Path(args.output_dir)
    output_dir.mkdir(exist_ok=True)

    sprite_path = "../../assets/art/sprites/world/land/Cstline1.spr.png"

    print(f"Loading sprites from {sprite_path}...")
    sprites = extract_sprites(sprite_path, cols=4, rows=7)
    print(
        f"Extracted {len(sprites)} rows x {len(sprites[0])} cols = {len(sprites) * len(sprites[0])} sprites"
    )

    print("Analyzing edge connections (max edge length: 50.0)...")
    connections, all_edges = analyze_connections(sprites, max_edge_length=40.0)
    print()

    print(f"Generating connection visualizations in {output_dir}/...")

    created_count = 0
    for row in range(len(sprites)):
        for col in range(len(sprites[0])):
            sprite_id_a = (row, col)
            sprite_a = sprites[row][col]

            vis = create_sprite_connections_visualization(
                sprite_a,
                sprites,
                sprite_id_a,
                connections,
                all_edges,
                max_per_edge=args.max_per_edge,
            )

            if vis is not None:
                print(f"  Creating visualization for Sprite ({row},{col})...")
                output_file = output_dir / f"sprite_{row}_{col}_connections.png"
                vis.save(output_file)
                created_count += 1

    print(
        f"\nDone! Generated {created_count} sprite connection visualizations in {output_dir}/"
    )


if __name__ == "__main__":
    main()
