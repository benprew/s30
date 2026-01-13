#!/usr/bin/env python3
"""
Analyze transition sprite edges to determine which sprites can connect together.

This script analyzes the Cstline1.spr.png sprite sheet to find matching edges
at corners, which indicates which transition sprites can be placed adjacent
to each other in the tile map.
"""

import argparse
import json
from PIL import Image
import numpy as np
from typing import List, Tuple, Dict
from dataclasses import dataclass


@dataclass
class EdgeSignature:
    """Represents an edge pattern at a corner"""

    direction: str  # e.g., "0->1", "3->2"
    length: int  # Number of non-transparent pixels
    pixels: List[Tuple[int, int]]  # Edge pixel coordinates

    def __len__(self):
        return self.length


# Corner positions for 102x52 sprites (hex tile shape)
CORNER_POSITIONS = {
    0: (51, 0),  # Top
    1: (101, 25),  # Right
    2: (51, 51),  # Bottom
    3: (0, 25),  # Left
}

# Valid edge directions (corner -> adjacent corner, both directions)
EDGE_DIRECTIONS = [
    (0, 1),  # Top to Right
    (1, 0),  # Right to Top
    (1, 2),  # Right to Bottom
    (2, 1),  # Bottom to Right
    (2, 3),  # Bottom to Left
    (3, 2),  # Left to Bottom
    (3, 0),  # Left to Top
    (0, 3),  # Top to Left
]

# Edge compatibility table
COMPATIBLE_EDGES = {
    (3, 2): (0, 1),
    (0, 1): (3, 2),
    (1, 2): (0, 3),
    (0, 3): (1, 2),
    (2, 1): (3, 0),
    (3, 0): (2, 1),
    (1, 0): (2, 3),
    (2, 3): (1, 0),
}

EDGE_TYPE_THRESHOLD = 40.0


def extract_sprites(
    image_path: str, cols: int, rows: int, total_rows: int = 21
) -> List[List[Image.Image]]:
    """Extract individual sprites from sprite sheet."""
    sheet = Image.open(image_path)
    width, height = sheet.size

    sprite_width = width // cols
    sprite_height = height // total_rows  # Total rows in full sheet (21)

    sprites = []
    for row in range(rows):
        sprite_row = []
        for col in range(cols):
            x = col * sprite_width
            y = row * sprite_height
            sprite = sheet.crop((x, y, x + sprite_width, y + sprite_height))
            sprite_row.append(sprite)
        sprites.append(sprite_row)

    return sprites


def get_line_pixels(x1: int, y1: int, x2: int, y2: int) -> List[Tuple[int, int]]:
    """Get all pixels along a line using Bresenham's algorithm."""
    pixels = []
    dx = abs(x2 - x1)
    dy = abs(y2 - y1)
    sx = 1 if x1 < x2 else -1
    sy = 1 if y1 < y2 else -1
    err = dx - dy

    x, y = x1, y1
    while True:
        pixels.append((x, y))
        if x == x2 and y == y2:
            break
        e2 = 2 * err
        if e2 > -dy:
            err -= dy
            x += sx
        if e2 < dx:
            err += dx
            y += sy

    return pixels


def extract_edge(
    sprite: Image.Image, from_corner: int, to_corner: int, width: int = 0
) -> EdgeSignature:
    """
    Extract edge from one corner to another by counting non-transparent pixels.
    Only returns valid edges that actually start from the "from" corner.

    Args:
        sprite: Sprite image
        from_corner: Starting corner (0-3)
        to_corner: Ending corner (0-3)
        width: Width of the band to check on either side of the line (default: 3)

    Returns:
        EdgeSignature with edge information
    """
    # Get corner positions
    x1, y1 = CORNER_POSITIONS[from_corner]
    x2, y2 = CORNER_POSITIONS[to_corner]

    # Get all pixels along the center line
    line_pixels = get_line_pixels(x1, y1, x2, y2)

    # Convert to RGBA to check alpha channel
    sprite_rgba = sprite.convert("RGBA")
    pixels_array = np.array(sprite_rgba)

    # Calculate perpendicular direction for width
    dx = x2 - x1
    dy = y2 - y1
    length = (dx * dx + dy * dy) ** 0.5
    if length == 0:
        perp_x, perp_y = 0, 0
    else:
        perp_x = -dy / length
        perp_y = dx / length

    # Count non-transparent pixels along the line with width
    edge_pixels = []
    seen_positions = set()

    for x, y in line_pixels:
        # Check pixels in a band around the line
        for offset in range(-width, width + 1):
            check_x = int(x + offset * perp_x)
            check_y = int(y + offset * perp_y)

            if (check_x, check_y) in seen_positions:
                continue
            seen_positions.add((check_x, check_y))

            if 0 <= check_x < sprite_rgba.width and 0 <= check_y < sprite_rgba.height:
                alpha = pixels_array[check_y, check_x, 3]
                if alpha > 128:  # Non-transparent threshold
                    edge_pixels.append((check_x, check_y))

    # Check if edge actually starts from the "from" corner
    # First few pixels from the starting corner should be non-transparent
    if len(edge_pixels) > 0:
        # Check if the first non-transparent pixel is within the first 5 pixels from start
        first_non_transparent_idx = None
        for i, (x, y) in enumerate(line_pixels[: min(10, len(line_pixels))]):
            # Check band around this position
            found = False
            for offset in range(-width, width + 1):
                check_x = int(x + offset * perp_x)
                check_y = int(y + offset * perp_y)
                if (
                    0 <= check_x < sprite_rgba.width
                    and 0 <= check_y < sprite_rgba.height
                ):
                    alpha = pixels_array[check_y, check_x, 3]
                    if alpha > 128:
                        found = True
                        break
            if found:
                first_non_transparent_idx = i
                break

        # If first non-transparent pixel is too far from start, this edge doesn't start here
        if first_non_transparent_idx is None or first_non_transparent_idx > 5:
            edge_pixels = []

    direction = f"{from_corner}->{to_corner}"
    length = len(edge_pixels)

    return EdgeSignature(direction=direction, length=length, pixels=edge_pixels)


def extract_all_edges(
    sprite: Image.Image, min_length: int = 10, width: int = 0
) -> Dict[str, EdgeSignature]:
    """Extract all possible edges from a sprite."""
    edges = {}
    for from_corner, to_corner in EDGE_DIRECTIONS:
        edge = extract_edge(sprite, from_corner, to_corner, width=width)
        if edge.length >= min_length:  # Only keep edges with minimum length
            edges[edge.direction] = edge
    return edges


def are_edges_compatible(dir1: str, dir2: str) -> bool:
    """Check if two edge directions are compatible for connection."""
    # Parse directions
    parts1 = dir1.split("->")
    parts2 = dir2.split("->")
    edge1 = (int(parts1[0]), int(parts1[1]))
    edge2 = (int(parts2[0]), int(parts2[1]))

    return COMPATIBLE_EDGES.get(edge1) == edge2


def calculate_similarity(edge1: EdgeSignature, edge2: EdgeSignature) -> float:
    """Calculate similarity score between two edges (0-100)."""
    if edge1.length == 0 or edge2.length == 0:
        return 0.0

    # Similarity based on length difference
    len_diff = abs(edge1.length - edge2.length)
    max_len = max(edge1.length, edge2.length)

    if max_len == 0:
        return 0.0

    # Percentage similarity (100% if same length, decreases with difference)
    similarity = max(0, 100 * (1 - len_diff / max_len))
    return similarity


def analyze_connections(
    sprites: List[List[Image.Image]], width: int = 0, edge_threshold: float = 40.0
) -> Tuple[Dict, Dict]:
    """Analyze all sprites to find connections based on matching edges."""
    rows = len(sprites)
    cols = len(sprites[0])

    # Extract all edges for all sprites
    all_edges = {}
    for row in range(rows):
        for col in range(cols):
            sprite = sprites[row][col]
            edges = extract_all_edges(sprite, width=width)
            all_edges[(row, col)] = edges

    # Find connections
    connections = {}
    for row in range(rows):
        for col in range(cols):
            sprite_id = (row, col)
            sprite_edges = all_edges[sprite_id]
            connections[sprite_id] = {}

            for direction, edge1 in sprite_edges.items():
                # Skip edges longer than edge_threshold
                if edge1.length > edge_threshold:
                    continue

                matches = []

                # Compare with all other sprites
                for other_row in range(rows):
                    for other_col in range(cols):
                        other_id = (other_row, other_col)
                        other_edges = all_edges[other_id]

                        # Check each edge of the other sprite
                        for other_direction, edge2 in other_edges.items():
                            # Skip edges longer than edge_threshold
                            if edge2.length > edge_threshold:
                                continue

                            # Only compare compatible directions
                            if are_edges_compatible(direction, other_direction):
                                similarity = calculate_similarity(edge1, edge2)
                                if similarity > 50:  # Threshold for match
                                    matches.append(
                                        (
                                            other_id,
                                            other_direction,
                                            edge2.length,
                                            similarity,
                                        )
                                    )

                # Sort by similarity
                matches.sort(key=lambda x: x[3], reverse=True)
                connections[sprite_id][direction] = (edge1.length, matches)

    return connections, all_edges


def print_connections(connections: Dict, all_edges: Dict, show_pixels: bool = False):
    """Print connection map in human-readable format."""
    print("Water Transition Sprite Connections")
    print("=" * 70)
    print()

    for sprite_id in sorted(connections.keys()):
        row, col = sprite_id
        has_connections = any(
            len(connections[sprite_id][d][1]) > 0 for d in connections[sprite_id]
        )

        if not has_connections:
            continue

        print(f"Sprite ({row},{col}):")

        sprite_edges = all_edges[sprite_id]
        for direction in sorted(connections[sprite_id].keys()):
            edge_length, matches = connections[sprite_id][direction]
            edge = sprite_edges[direction]

            print(f"  Edge [{direction}] (length: {edge_length}):")

            if show_pixels and edge.pixels:
                first = edge.pixels[0]
                last = edge.pixels[-1]
                print(f"    Pixels: {first} -> {last}")

            if matches:
                for other_id, other_direction, other_length, similarity in matches[:5]:
                    other_row, other_col = other_id
                    print(
                        f"    Connects to: Sprite ({other_row},{other_col}) [{other_direction}] (length: {other_length}) (similarity: {similarity:.1f}%)"
                    )

                    if show_pixels:
                        other_edge = all_edges[other_id][other_direction]
                        if other_edge.pixels:
                            other_first = other_edge.pixels[0]
                            other_last = other_edge.pixels[-1]
                            print(f"      Pixels: {other_first} -> {other_last}")

        print()


def generate_tile_map(
    connections: Dict, all_edges: Dict, edge_threshold: float = 40.0
) -> Dict:
    """Generate simplified tile map with full edges and connection edges."""
    tile_map = {}

    for sprite_id in sorted(connections.keys()):
        row, col = sprite_id
        tile_key = f"{row},{col}"

        sprite_edges = all_edges[sprite_id]

        full_edges = []
        connect_edges = {}

        for direction, edge in sprite_edges.items():
            if edge.length >= edge_threshold:
                full_edges.append(direction)
            elif direction in connections[sprite_id]:
                edge_length, matches = connections[sprite_id][direction]
                if matches:
                    connect_edges[direction] = [
                        f"{m[0][0]},{m[0][1]}" for m in matches
                    ]

        tile_map[tile_key] = {"full": full_edges, "connect": connect_edges}

    return tile_map


def main():
    parser = argparse.ArgumentParser(
        description="Analyze transition sprite edge connections"
    )
    parser.add_argument(
        "--show-pixels", action="store_true", help="Show pixel coordinates for edges"
    )
    parser.add_argument("--debug", action="store_true", help="Show debug output")
    parser.add_argument(
        "--output-json", type=str, help="Output tile map as JSON to specified file"
    )

    args = parser.parse_args()

    sprite_path = "../../assets/art/sprites/world/land/Cstline1.spr.png"

    print(f"Loading sprites from {sprite_path}...")
    sprites = extract_sprites(sprite_path, cols=4, rows=7)
    print(
        f"Extracted {len(sprites)} rows x {len(sprites[0])} cols = {len(sprites) * len(sprites[0])} water transition sprites"
    )
    print()

    if args.debug:
        print("Debug: Detailed analysis of sprite (0,0) edge 3->0 with widened band:")
        print("=" * 70)
        sprite00 = sprites[0][0]
        sprite_rgba = sprite00.convert("RGBA")
        pixels_array = np.array(sprite_rgba)

        # Check edge 3->0
        x1, y1 = CORNER_POSITIONS[3]  # (0, 19)
        x2, y2 = CORNER_POSITIONS[0]  # (51, 0)
        print(f"Corner 3: {(x1, y1)}, Corner 0: {(x2, y2)}")

        line_pixels = get_line_pixels(x1, y1, x2, y2)
        print(f"Total pixels along center line: {len(line_pixels)}")

        # Calculate perpendicular direction
        dx = x2 - x1
        dy = y2 - y1
        length = (dx * dx + dy * dy) ** 0.5
        perp_x = -dy / length
        perp_y = dx / length
        width = 0

        print(
            f"\nChecking first 10 positions along line 3->0 (with width={width} band):"
        )
        for i, (x, y) in enumerate(line_pixels[:10]):
            found_non_transparent = False
            for offset in range(-width, width + 1):
                check_x = int(x + offset * perp_x)
                check_y = int(y + offset * perp_y)
                if (
                    0 <= check_x < sprite_rgba.width
                    and 0 <= check_y < sprite_rgba.height
                ):
                    alpha = pixels_array[check_y, check_x, 3]
                    if alpha > 128:
                        found_non_transparent = True
                        print(
                            f"  Position {i}: center({x}, {y}) -> FOUND non-transparent at offset {offset}: ({check_x}, {check_y}) alpha={alpha}"
                        )
                        break
            if not found_non_transparent:
                print(f"  Position {i}: center({x}, {y}) -> all transparent in band")

        print()
        print("=" * 70)
        print()

        print("Debug: Checking ALL edges for first row (row 0) sprites (width=0):")
        print("=" * 70)
        for col in range(4):
            sprite = sprites[0][col]
            print(f"\nSprite (0,{col}):")
            # Extract all edges without minimum length filter
            for from_corner, to_corner in EDGE_DIRECTIONS:
                edge = extract_edge(sprite, from_corner, to_corner, width=0)
                print(f"  Edge {from_corner}->{to_corner}: {edge.length} pixels")
                if edge.pixels and edge.length > 0:
                    print(f"    From {edge.pixels[0]} to {edge.pixels[-1]}")
        print()
        print("=" * 70)
        print()

    print(f"Analyzing edge connections (edge threshold: {EDGE_TYPE_THRESHOLD})...")
    connections, all_edges = analyze_connections(
        sprites, edge_threshold=EDGE_TYPE_THRESHOLD
    )
    print()

    if args.output_json:
        tile_map = generate_tile_map(connections, all_edges, EDGE_TYPE_THRESHOLD)
        with open(args.output_json, "w") as f:
            json.dump(tile_map, f, indent=2)
        print(f"Tile map written to {args.output_json}")
    else:
        print_connections(connections, all_edges, args.show_pixels)


if __name__ == "__main__":
    main()
