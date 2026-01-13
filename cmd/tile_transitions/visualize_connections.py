#!/usr/bin/env python3
"""
Visualize sprite connections by drawing edges and labeling them with potential matches.
"""

import argparse
from PIL import Image, ImageDraw, ImageFont
import numpy as np
from pathlib import Path
from analyze_edges import (
    extract_sprites, extract_all_edges, analyze_connections,
    CORNER_POSITIONS, COMPATIBLE_EDGES, are_edges_compatible
)


def draw_corner_markers(draw, sprite_width, sprite_height, radius=3):
    """Draw markers at each corner position."""
    colors = {
        0: (255, 0, 0),    # Red - top
        1: (0, 255, 0),    # Green - right
        2: (0, 0, 255),    # Blue - bottom
        3: (255, 255, 0),  # Yellow - left
    }

    for corner_id, (x, y) in CORNER_POSITIONS.items():
        color = colors[corner_id]
        draw.ellipse([x-radius, y-radius, x+radius, y+radius], fill=color, outline=(255,255,255))


def draw_edge_line(draw, from_corner, to_corner, color=(255, 0, 255), width=2):
    """Draw a line representing an edge."""
    x1, y1 = CORNER_POSITIONS[from_corner]
    x2, y2 = CORNER_POSITIONS[to_corner]
    draw.line([x1, y1, x2, y2], fill=color, width=width)


def create_sprite_visualization(sprite_img, sprite_id, edges, connections):
    """Create visualization for a single sprite showing its edges and connections."""
    # Create larger canvas for labels
    margin = 150
    canvas_width = sprite_img.width + 2 * margin
    canvas_height = sprite_img.height + 2 * margin

    canvas = Image.new('RGBA', (canvas_width, canvas_height), (255, 255, 255, 255))
    draw = ImageDraw.Draw(canvas)

    # Paste sprite in center
    sprite_x = margin
    sprite_y = margin
    canvas.paste(sprite_img, (sprite_x, sprite_y))

    # Draw corner markers on the sprite
    draw_shifted = ImageDraw.Draw(canvas)
    for corner_id, (x, y) in CORNER_POSITIONS.items():
        color_map = {0: (255, 0, 0), 1: (0, 255, 0), 2: (0, 0, 255), 3: (255, 255, 0)}
        color = color_map[corner_id]
        draw_shifted.ellipse([
            sprite_x + x - 3, sprite_y + y - 3,
            sprite_x + x + 3, sprite_y + y + 3
        ], fill=color, outline=(0, 0, 0))

    # Draw edges
    edge_colors = {
        '0->1': (255, 100, 100),
        '1->0': (255, 150, 150),
        '1->2': (100, 255, 100),
        '2->1': (150, 255, 150),
        '2->3': (100, 100, 255),
        '3->2': (150, 150, 255),
        '3->0': (255, 255, 100),
        '0->3': (255, 255, 150),
    }

    y_offset = 10

    for direction, edge in edges.items():
        from_corner = int(direction.split('->')[0])
        to_corner = int(direction.split('->')[1])

        color = edge_colors.get(direction, (255, 0, 255))

        # Draw edge line
        x1, y1 = CORNER_POSITIONS[from_corner]
        x2, y2 = CORNER_POSITIONS[to_corner]
        draw_shifted.line([
            sprite_x + x1, sprite_y + y1,
            sprite_x + x2, sprite_y + y2
        ], fill=color, width=3)

        # Add label
        if direction in connections[sprite_id]:
            edge_length, matches = connections[sprite_id][direction]
            label = f"Edge {direction} ({edge_length}px):"
            draw.text((10, y_offset), label, fill=(0, 0, 0))
            y_offset += 20

            # Show top 3 matches
            for i, (other_id, other_dir, other_len, similarity) in enumerate(matches[:3]):
                match_text = f"  → Sprite{other_id} [{other_dir}] ({other_len}px) {similarity:.1f}%"
                draw.text((10, y_offset), match_text, fill=(100, 100, 100))
                y_offset += 15

            y_offset += 5

    # Draw title
    title = f"Sprite {sprite_id}"
    draw.text((canvas_width // 2 - 50, 5), title, fill=(0, 0, 0))

    return canvas


def main():
    parser = argparse.ArgumentParser(description='Visualize sprite edge connections')
    parser.add_argument('--output-dir', default='sprite_visualizations',
                       help='Directory to save visualization images')
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

    print(f"Generating visualizations in {output_dir}/...")

    for row in range(len(sprites)):
        for col in range(len(sprites[0])):
            sprite_id = (row, col)
            sprite = sprites[row][col]
            edges = all_edges[sprite_id]

            # Only create visualization if sprite has edges
            if len(edges) == 0:
                continue

            # Check if it has any connections
            has_connections = any(
                len(connections[sprite_id].get(direction, (0, []))[1]) > 0
                for direction in edges.keys()
            )

            if not has_connections:
                continue

            print(f"  Creating visualization for Sprite ({row},{col})...")

            vis = create_sprite_visualization(sprite, sprite_id, edges, connections)

            output_file = output_dir / f"sprite_{row}_{col}.png"
            vis.save(output_file)

    print(f"\nDone! Visualizations saved to {output_dir}/")


if __name__ == '__main__':
    main()
