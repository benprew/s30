#!/usr/bin/env python3
"""
Visualize sprite connection pairs by showing connected sprites side-by-side.
"""

import argparse
from PIL import Image, ImageDraw, ImageFont
import numpy as np
from pathlib import Path
from analyze_edges import (
    extract_sprites, extract_all_edges, analyze_connections,
    CORNER_POSITIONS, COMPATIBLE_EDGES
)


def get_sprite_offset_for_connection(from_edge, to_edge):
    """
    Calculate the offset needed to position sprite B relative to sprite A
    so that their connecting edges align.

    Args:
        from_edge: tuple (from_corner, to_corner) for sprite A
        to_edge: tuple (from_corner, to_corner) for sprite B

    Returns:
        (offset_x, offset_y) to position sprite B relative to sprite A
    """
    # Get the corners that need to align
    # Sprite A's to_corner should align with sprite B's from_corner
    # Sprite A's from_corner should align with sprite B's to_corner

    from_corner_a, to_corner_a = from_edge
    from_corner_b, to_corner_b = to_edge

    # Get positions for sprite A's edge endpoints
    x1_a, y1_a = CORNER_POSITIONS[from_corner_a]
    x2_a, y2_a = CORNER_POSITIONS[to_corner_a]

    # Get positions for sprite B's edge endpoints
    x1_b, y1_b = CORNER_POSITIONS[from_corner_b]
    x2_b, y2_b = CORNER_POSITIONS[to_corner_b]

    # Sprite B should be positioned so that:
    # B's from_corner aligns with A's to_corner
    # B's to_corner aligns with A's from_corner

    # Calculate offset to align B's from_corner with A's to_corner
    offset_x = x2_a - x1_b
    offset_y = y2_a - y1_b

    return (offset_x, offset_y)


def create_connection_pair_visualization(sprite_a, sprite_b, sprite_id_a, sprite_id_b,
                                         edge_a, edge_b, similarity):
    """
    Create visualization showing two sprites connected by their matching edges.
    """
    # Parse edge directions
    from_a, to_a = map(int, edge_a.split('->'))
    from_b, to_b = map(int, edge_b.split('->'))

    # Calculate positioning
    offset_x, offset_y = get_sprite_offset_for_connection((from_a, to_a), (from_b, to_b))

    # Create canvas large enough to hold both sprites
    margin = 50
    canvas_width = sprite_a.width * 2 + abs(offset_x) + margin * 2
    canvas_height = sprite_a.height * 2 + abs(offset_y) + margin * 2

    canvas = Image.new('RGBA', (canvas_width, canvas_height), (240, 240, 240, 255))
    draw = ImageDraw.Draw(canvas)

    # Position sprite A at a reference point
    sprite_a_x = margin + (sprite_a.width if offset_x < 0 else 0)
    sprite_a_y = margin + (sprite_a.height if offset_y < 0 else 0)

    # Position sprite B based on offset
    sprite_b_x = sprite_a_x + offset_x
    sprite_b_y = sprite_a_y + offset_y

    # Draw semi-transparent background boxes
    draw.rectangle([sprite_a_x-2, sprite_a_y-2,
                   sprite_a_x+sprite_a.width+2, sprite_a_y+sprite_a.height+2],
                  outline=(200, 0, 0), width=2)
    draw.rectangle([sprite_b_x-2, sprite_b_y-2,
                   sprite_b_x+sprite_b.width+2, sprite_b_y+sprite_b.height+2],
                  outline=(0, 0, 200), width=2)

    # Paste sprites (convert to RGBA for transparency)
    sprite_a_rgba = sprite_a.convert('RGBA')
    sprite_b_rgba = sprite_b.convert('RGBA')
    canvas.paste(sprite_a_rgba, (sprite_a_x, sprite_a_y), sprite_a_rgba)
    canvas.paste(sprite_b_rgba, (sprite_b_x, sprite_b_y), sprite_b_rgba)

    # Draw the connecting edge on sprite A
    x1_a, y1_a = CORNER_POSITIONS[from_a]
    x2_a, y2_a = CORNER_POSITIONS[to_a]
    draw.line([sprite_a_x + x1_a, sprite_a_y + y1_a,
               sprite_a_x + x2_a, sprite_a_y + y2_a],
              fill=(255, 0, 0), width=3)

    # Draw the connecting edge on sprite B
    x1_b, y1_b = CORNER_POSITIONS[from_b]
    x2_b, y2_b = CORNER_POSITIONS[to_b]
    draw.line([sprite_b_x + x1_b, sprite_b_y + y1_b,
               sprite_b_x + x2_b, sprite_b_y + y2_b],
              fill=(0, 0, 255), width=3)

    # Draw corner markers
    corner_colors = {0: (255, 0, 0), 1: (0, 255, 0), 2: (0, 0, 255), 3: (255, 255, 0)}

    for corner_id, (x, y) in CORNER_POSITIONS.items():
        # Sprite A corners
        draw.ellipse([sprite_a_x + x - 3, sprite_a_y + y - 3,
                     sprite_a_x + x + 3, sprite_a_y + y + 3],
                    fill=corner_colors[corner_id], outline=(0, 0, 0))
        # Sprite B corners
        draw.ellipse([sprite_b_x + x - 3, sprite_b_y + y - 3,
                     sprite_b_x + x + 3, sprite_b_y + y + 3],
                    fill=corner_colors[corner_id], outline=(0, 0, 0))

    # Add labels
    title = f"Sprite {sprite_id_a} [{edge_a}] ← ({similarity:.1f}%) → Sprite {sprite_id_b} [{edge_b}]"
    draw.text((10, 10), title, fill=(0, 0, 0))

    # Label sprites
    draw.text((sprite_a_x + 5, sprite_a_y + 5), f"({sprite_id_a[0]},{sprite_id_a[1]})",
              fill=(200, 0, 0))
    draw.text((sprite_b_x + 5, sprite_b_y + 5), f"({sprite_id_b[0]},{sprite_id_b[1]})",
              fill=(0, 0, 200))

    return canvas


def main():
    parser = argparse.ArgumentParser(description='Visualize sprite connection pairs')
    parser.add_argument('--output-dir', default='sprite_pairs',
                       help='Directory to save pair visualizations')
    parser.add_argument('--max-per-sprite', type=int, default=3,
                       help='Maximum connections to visualize per sprite edge')
    args = parser.parse_args()

    # Create output directory
    output_dir = Path(args.output_dir)
    output_dir.mkdir(exist_ok=True)

    sprite_path = '../../assets/art/sprites/world/land/Cstline1.spr.png'

    print(f"Loading sprites from {sprite_path}...")
    sprites = extract_sprites(sprite_path, cols=4, rows=7)
    print(f"Extracted {len(sprites)} rows x {len(sprites[0])} cols = {len(sprites) * len(sprites[0])} sprites")

    print("Analyzing edge connections...")
    connections, all_edges = analyze_connections(sprites)
    print()

    print(f"Generating pair visualizations in {output_dir}/...")

    pair_count = 0
    for row in range(len(sprites)):
        for col in range(len(sprites[0])):
            sprite_id_a = (row, col)
            sprite_a = sprites[row][col]
            edges_a = all_edges[sprite_id_a]

            if len(edges_a) == 0:
                continue

            for direction_a, edge_a in edges_a.items():
                if direction_a not in connections[sprite_id_a]:
                    continue

                edge_length, matches = connections[sprite_id_a][direction_a]

                if len(matches) == 0:
                    continue

                # Create visualization for top N matches
                for i, (sprite_id_b, direction_b, length_b, similarity) in enumerate(matches[:args.max_per_sprite]):
                    row_b, col_b = sprite_id_b
                    sprite_b = sprites[row_b][col_b]

                    print(f"  Creating pair: Sprite ({row},{col}) [{direction_a}] ← → Sprite {sprite_id_b} [{direction_b}] ({similarity:.1f}%)")

                    vis = create_connection_pair_visualization(
                        sprite_a, sprite_b,
                        sprite_id_a, sprite_id_b,
                        direction_a, direction_b,
                        similarity
                    )

                    output_file = output_dir / f"pair_{row}_{col}_{direction_a.replace('->', '_')}__to__{row_b}_{col_b}_{direction_b.replace('->', '_')}.png"
                    vis.save(output_file)
                    pair_count += 1

    print(f"\nDone! Generated {pair_count} pair visualizations in {output_dir}/")


if __name__ == '__main__':
    main()
