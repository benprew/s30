#!/usr/bin/env python3
"""
Script to build Magic: The Gathering card images from Scryfall JSON data.
Composites Scryfall art crops (ArtURL) onto vintage card frames in assets/art/card/,
rendering rules text, casting costs, type lines, power/toughness, and artist credit.
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
from functools import lru_cache
from pathlib import Path
from typing import Any, Dict, List, Set

from PIL import Image, ImageDraw, ImageFont

try:
    import zstandard as zstd

    HAS_ZSTD = True
except ImportError:
    HAS_ZSTD = False

logger = logging.getLogger(__name__)

CARD_WIDTH = 245
JPEG_QUALITY = 80

# Mana symbol mapping in assets/art/card/Manasymbols.pic.png (19 icons of 18x18)
# Icon 0: X, Icon 1: 0, Icon 2..10: 1..9, Icon 11: 10
# Icon 12: W (Sun), Icon 13: R (Fire), Icon 14: U (Drop)
# Icon 15: B (Skull), Icon 16: G (Tree), Icon 17: T (Tap)
TOKEN_TO_MANA_ICON: Dict[str, int] = {
    "X": 0,
    "0": 1,
    "1": 2,
    "2": 3,
    "3": 4,
    "4": 5,
    "5": 6,
    "6": 7,
    "7": 8,
    "8": 9,
    "9": 10,
    "10": 11,
    "W": 12,
    "R": 13,
    "U": 14,
    "B": 15,
    "G": 16,
    "T": 17,
    "C": 1,
}

# Set symbol mapping in assets/art/card/Cardsets.pic.png (22 icons of 15x15)
SET_TO_SET_ICON: Dict[str, int] = {
    "arn": 1,
    "atq": 10,
    "leg": 9,
    "drk": 5,
    "fem": 6,
    "ice": 18,
}


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
    return f"{set_code}-{collector_number}-200-{sanitized_name}.jpg"


def get_frame_filename(card_data: Dict[str, Any]) -> str:
    """Determine the vintage card frame filename from card attributes."""
    type_line = card_data.get("TypeLine", "") or ""
    colors = card_data.get("Colors", []) or []
    set_id = (card_data.get("SetID") or "").lower()

    if "Land" in type_line:
        # Check basic lands
        if "Plains" in type_line and not any(
            k in type_line for k in ["Island", "Swamp", "Mountain", "Forest"]
        ):
            return "Cardbk_Whiteland.pic.png"
        if "Island" in type_line and not any(
            k in type_line for k in ["Plains", "Swamp", "Mountain", "Forest"]
        ):
            return "Cardbk_Blueland.pic.png"
        if "Swamp" in type_line and not any(
            k in type_line for k in ["Plains", "Island", "Mountain", "Forest"]
        ):
            return "Cardbk_Blackland.pic.png"
        if "Mountain" in type_line and not any(
            k in type_line for k in ["Plains", "Island", "Swamp", "Forest"]
        ):
            return "Cardbk_Redland.pic.png"
        if "Forest" in type_line and not any(
            k in type_line for k in ["Plains", "Island", "Swamp", "Mountain"]
        ):
            return "Cardbk_Greenland.pic.png"

        # Expansion non-basic lands
        if set_id == "atq":
            return "Cardbk_Antiquitiesland.pic.png"
        if set_id == "arn":
            return "Cardbk_Arabiannightsland.pic.png"
        if set_id == "drk":
            return "Cardbk_Darklandsland.pic.png"
        if set_id == "fem":
            return "Cardbk_Fallenempiresland.pic.png"
        if set_id == "leg":
            return "Cardbk_Legendsland.pic.png"
        if set_id == "ice":
            return "Cardbk_Iceageland.pic.png"
        return "Cardbk_Antiquitiesland.pic.png"

    if "Artifact" in type_line:
        return "Cardbk_Artifact.pic.png"

    if len(colors) > 1:
        return "Cardbk_Gold.pic.png"

    if len(colors) == 1:
        c = colors[0]
        if c == "W":
            return "Cardbk_White.pic.png"
        if c == "U":
            return "Cardbk_Blue.pic.png"
        if c == "B":
            return "Cardbk_Black.pic.png"
        if c == "R":
            return "Cardbk_Red.pic.png"
        if c == "G":
            return "Cardbk_Green.pic.png"

    return "Cardbk_Special.pic.png"


@lru_cache(maxsize=1)
def load_mana_icons(assets_dir: Path) -> Dict[int, Image.Image]:
    """Load and prepare transparent circular mana symbol icons from sprite sheet."""
    mana_file = assets_dir / "art" / "card" / "Manasymbols.pic.png"
    mana_sheet = Image.open(mana_file).convert("RGBA")
    icons: Dict[int, Image.Image] = {}

    for i in range(19):
        raw_icon = mana_sheet.crop((i * 18, 0, (i + 1) * 18, 18))
        icon_rgba = Image.new("RGBA", (18, 18), (0, 0, 0, 0))
        for y in range(18):
            for x in range(18):
                px = raw_icon.getpixel((x, y))
                if isinstance(px, tuple) and len(px) >= 3:
                    r, g, b = int(px[0]), int(px[1]), int(px[2])
                    dx = x - 8.5
                    dy = y - 8.5
                    dist = dx * dx + dy * dy
                    if dist <= 64 and not (r == 0 and g == 0 and b == 0):
                        icon_rgba.putpixel((x, y), (r, g, b, 255))
                    elif dist <= 72 and not (r == 0 and g == 0 and b == 0):
                        alpha = int(255 * (1 - (dist - 64) / 8))
                        icon_rgba.putpixel((x, y), (r, g, b, alpha))
        icons[i] = icon_rgba

    return icons


@lru_cache(maxsize=1)
def load_set_icons(assets_dir: Path) -> Dict[int, Image.Image]:
    """Load set expansion icons with transparent backgrounds."""
    set_file = assets_dir / "art" / "card" / "Cardsets.pic.png"
    set_sheet = Image.open(set_file).convert("RGBA")
    icons: Dict[int, Image.Image] = {}

    for i in range(22):
        raw_s = set_sheet.crop((i * 15, 0, (i + 1) * 15, 15))
        s_rgba = Image.new("RGBA", (15, 15), (0, 0, 0, 0))
        for y in range(15):
            for x in range(15):
                px = raw_s.getpixel((x, y))
                if isinstance(px, tuple) and len(px) >= 3:
                    r, g, b = int(px[0]), int(px[1]), int(px[2])
                    if not (r == 0 and g == 0 and b == 0):
                        s_rgba.putpixel((x, y), (r, g, b, 255))
        icons[i] = s_rgba

    return icons


@lru_cache(maxsize=32)
def get_font(font_path: Path, size: int) -> ImageFont.FreeTypeFont:
    """Cached font loader."""
    return ImageFont.truetype(str(font_path), size)


def wrap_text(
    txt: str, font: ImageFont.FreeTypeFont, max_w: int, draw: ImageDraw.ImageDraw
) -> List[str]:
    """Wrap text into lines fitting within max_w pixels."""
    lines: List[str] = []
    for para in txt.split("\n"):
        words = para.split(" ")
        curr_line = ""
        for w in words:
            test_line = f"{curr_line} {w}".strip()
            clean_test = re.sub(r"\{[^}]+\}", "MM", test_line)
            bbox = draw.textbbox((0, 0), clean_test, font=font)
            if (bbox[2] - bbox[0]) <= max_w:
                curr_line = test_line
            else:
                if curr_line:
                    lines.append(curr_line)
                curr_line = w
        if curr_line:
            lines.append(curr_line)
    return lines


def build_card_image(
    card_data: Dict[str, Any],
    art_image: Image.Image | None,
    assets_dir: Path,
) -> Image.Image:
    """Build a complete 228x325 card image from art and metadata."""
    frame_filename = get_frame_filename(card_data)
    frame_path = assets_dir / "art" / "card" / frame_filename
    card_img = Image.open(frame_path).convert("RGBA")

    is_black_frame = "Cardbk_Black.pic.png" in frame_filename
    header_text_color = (255, 255, 255, 255) if is_black_frame else (0, 0, 0, 255)
    body_text_color = (0, 0, 0, 255)

    mana_icons = load_mana_icons(assets_dir)
    set_icons = load_set_icons(assets_dir)
    medieval_font_path = assets_dir / "fonts" / "Magim___.ttf"
    plane_font_path = assets_dir / "fonts" / "Planewalker-38m6.ttf"

    # 1. Art crop (21, 38, 207, 156) -> width=186, height=118
    art_w, art_h = 186, 118
    if art_image is not None:
        scale = max(art_w / art_image.width, art_h / art_image.height)
        new_w = max(art_w, int(art_image.width * scale))
        new_h = max(art_h, int(art_image.height * scale))
        resized_art = art_image.resize((new_w, new_h), Image.Resampling.LANCZOS)
        crop_x = (new_w - art_w) // 2
        crop_y = (new_h - art_h) // 2
        cropped_art = resized_art.crop((crop_x, crop_y, crop_x + art_w, crop_y + art_h))
        card_img.paste(cropped_art, (21, 38))

    draw = ImageDraw.Draw(card_img)

    # 2. Mana Cost (drawn right-to-left at x=204, y=18)
    mana_cost = card_data.get("ManaCost", "") or ""
    tokens = re.findall(r"\{([^}]+)\}", mana_cost)
    curr_x = 204
    mana_pip_size = 14
    for tok in reversed(tokens):
        icon_idx = TOKEN_TO_MANA_ICON.get(tok.upper())
        if icon_idx is not None:
            icon = mana_icons[icon_idx].resize(
                (mana_pip_size, mana_pip_size), Image.Resampling.LANCZOS
            )
            curr_x -= mana_pip_size
            card_img.paste(icon, (curr_x, 18), icon)
            curr_x -= 1

    # 3. Card Name (left aligned at x=24, y=19)
    name = card_data.get("CardName", "")
    max_name_w = curr_x - 26
    font_size = 13
    font_title = get_font(medieval_font_path, font_size)
    while font_size > 8:
        bbox = draw.textbbox((0, 0), name, font=font_title)
        if (bbox[2] - bbox[0]) <= max_name_w:
            break
        font_size -= 1
        font_title = get_font(medieval_font_path, font_size)
    draw.text((24, 19), name, font=font_title, fill=header_text_color)

    # 4. Type Line (x=24, y=163)
    type_line = card_data.get("TypeLine", "") or ""
    type_line = type_line.replace("\u2014", "-").replace("—", "-")
    font_type = get_font(medieval_font_path, 11)

    set_id = (card_data.get("SetID") or "").lower()
    set_idx = SET_TO_SET_ICON.get(set_id)
    if set_idx is not None:
        s_icon = set_icons[set_idx]
        card_img.paste(s_icon, (192, 161), s_icon)

    draw.text((24, 163), type_line, font=font_type, fill=header_text_color)

    # 5. Rules Text & Flavor Text
    text = card_data.get("Text", "") or ""
    flavor = card_data.get("FlavorText", "") or ""
    power = card_data.get("Power")
    toughness = card_data.get("Toughness")
    is_creature = power is not None and toughness is not None

    high_box_frames = [
        "Cardbk_Artifact.pic.png",
        "Cardbk_White.pic.png",
        "Cardbk_Iceageland.pic.png",
    ]
    y_text_start = 186 if any(k in frame_filename for k in high_box_frames) else 204
    max_text_h = 292 - y_text_start

    chosen_font_size = 10
    chosen_lines: List[str] = []
    chosen_flavor: List[str] = []
    chosen_line_h = 12

    for fsz in [10, 9, 8, 7]:
        f_text = get_font(plane_font_path, fsz)
        f_flav = get_font(plane_font_path, max(6, fsz - 1))
        lh = fsz + 2
        t_lines = wrap_text(text, f_text, 178, draw) if text else []
        fl_lines = wrap_text(flavor, f_flav, 178, draw) if flavor else []
        total_h = len(t_lines) * lh + (len(fl_lines) * (fsz + 1) + 4 if fl_lines else 0)

        if total_h <= max_text_h:
            chosen_font_size = fsz
            chosen_lines = t_lines
            chosen_flavor = fl_lines
            chosen_line_h = lh
            break
        elif len(t_lines) * lh <= max_text_h:
            chosen_font_size = fsz
            chosen_lines = t_lines
            chosen_flavor = []
            chosen_line_h = lh
            break
        else:
            chosen_font_size = fsz
            chosen_lines = t_lines
            chosen_flavor = []
            chosen_line_h = lh

    font_text = get_font(plane_font_path, chosen_font_size)
    font_flavor = get_font(plane_font_path, max(6, chosen_font_size - 1))

    y_text = y_text_start
    for line in chosen_lines:
        parts = re.split(r"(\{[^}]+\})", line)
        x_pos = 25
        for p in parts:
            if not p:
                continue
            m = re.match(r"^\{([^}]+)\}$", p)
            if m:
                tok = m.group(1).upper()
                icon_idx = TOKEN_TO_MANA_ICON.get(tok)
                if icon_idx is not None:
                    icon_sz = max(8, chosen_font_size)
                    icon = mana_icons[icon_idx].resize(
                        (icon_sz, icon_sz), Image.Resampling.LANCZOS
                    )
                    card_img.paste(icon, (int(x_pos), y_text + 1), icon)
                    x_pos += icon_sz + 1
                else:
                    draw.text(
                        (int(x_pos), y_text),
                        p,
                        font=font_text,
                        fill=body_text_color,
                    )
                    bbox = draw.textbbox((0, 0), p, font=font_text)
                    x_pos += int(bbox[2] - bbox[0])
            else:
                draw.text((int(x_pos), y_text), p, font=font_text, fill=body_text_color)
                bbox = draw.textbbox((0, 0), p, font=font_text)
                x_pos += int(bbox[2] - bbox[0])
        y_text += chosen_line_h

    if chosen_flavor and (y_text + len(chosen_flavor) * (chosen_font_size + 1) <= 292):
        y_text += 2
        for fline in chosen_flavor:
            draw.text((25, y_text), fline, font=font_flavor, fill=(70, 70, 70, 255))
            y_text += chosen_font_size + 1

    # 6. Power / Toughness Box for creatures
    if is_creature:
        pt_str = f"{power}/{toughness}"
        pt_box_x1, pt_box_y1 = 172, 294
        pt_box_x2, pt_box_y2 = 212, 313
        draw.rectangle(
            (pt_box_x1, pt_box_y1, pt_box_x2, pt_box_y2),
            fill=(240, 236, 226, 255),
            outline=(50, 40, 30, 255),
            width=1,
        )
        font_pt = get_font(medieval_font_path, 12)
        pt_bbox = draw.textbbox((0, 0), pt_str, font=font_pt)
        pt_w = pt_bbox[2] - pt_bbox[0]
        pt_x = pt_box_x1 + (pt_box_x2 - pt_box_x1 - pt_w) // 2
        draw.text((pt_x, pt_box_y1 + 2), pt_str, font=font_pt, fill=(0, 0, 0, 255))

    # 7. Artist credit
    artist = card_data.get("Artist", "")
    if artist:
        font_artist = get_font(plane_font_path, 7)
        credit_color = header_text_color if is_black_frame else (50, 50, 50, 255)
        draw.text((24, 306), f"Illus. {artist}", font=font_artist, fill=credit_color)

    return card_img


def download_and_process_card(
    card_data: Dict[str, Any],
    existing_files: Set[str],
    output_dir: Path,
    assets_dir: Path | None = None,
) -> tuple[bool, str]:
    """Download art and build a single card image, saving to output_dir."""
    if assets_dir is None:
        assets_dir = Path("assets")

    card_name = card_data.get("CardName", "")
    try:
        set_code = card_data.get("SetID", "")
        collector_number = card_data.get("CollectorNo", "")
        art_url = card_data.get("ArtURL") or card_data.get("BorderCropURL")

        if not art_url:
            logger.error(f"No art URL found for {card_name}")
            return False, ""

        resized_filename = image_filename(card_data)

        if resized_filename in existing_files:
            logger.info(
                f"Skipping {card_name} ({set_code}-{collector_number}) "
                "- already exists in zip"
            )
            return True, ""

        logger.info(f"Processing {card_name} ({set_code}-{collector_number})")
        logger.info(f"Downloading art from {art_url}")

        req = urllib.request.Request(
            art_url,
            headers={
                "User-Agent": (
                    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
                    "AppleWebKit/537.36 (KHTML, like Gecko) "
                    "Chrome/120.0.0.0 Safari/537.36"
                ),
                "Accept": (
                    "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8"
                ),
            },
        )
        with urllib.request.urlopen(req, timeout=30) as response:
            image_data = response.read()

        with Image.open(io.BytesIO(image_data)) as art_img:
            art_rgb = art_img.convert("RGB")
            built_card = build_card_image(card_data, art_rgb, assets_dir)

            output_path = output_dir / resized_filename
            target_height = round(built_card.height * CARD_WIDTH / built_card.width)
            resized = built_card.resize(
                (CARD_WIDTH, target_height), Image.Resampling.LANCZOS
            )
            resized.convert("RGB").save(
                output_path,
                format="JPEG",
                quality=JPEG_QUALITY,
                optimize=True,
                progressive=True,
                subsampling="4:2:0",
            )

        print(f"Completed: {card_name} ({set_code}-{collector_number})")
        return True, resized_filename

    except Exception as e:
        logger.error(f"Error processing {card_name}: {e}")
        return False, ""


def retain_archive_files(zip_path: Path, filenames: Set[str]) -> None:
    """Atomically remove archive entries that are not in filenames."""
    temporary_path = zip_path.with_suffix(f"{zip_path.suffix}.tmp")
    try:
        with zipfile.ZipFile(zip_path, "r") as source:
            with zipfile.ZipFile(temporary_path, "w") as destination:
                for filename in sorted(filenames):
                    destination.writestr(filename, source.read(filename))
        temporary_path.replace(zip_path)
    finally:
        temporary_path.unlink(missing_ok=True)


def main() -> None:
    parser = argparse.ArgumentParser(
        description=(
            "Download art and build Magic: The Gathering card images "
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
    parser.add_argument(
        "--assets-dir",
        default="assets",
        help="Path to assets directory containing fonts and card frames",
    )
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO if args.verbose else logging.WARNING,
        format="%(message)s",
    )

    assets_dir = Path(args.assets_dir)
    zip_path = assets_dir / "art" / "cardimages.zip"
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
                    download_and_process_card,
                    card,
                    existing_files,
                    output_dir,
                    assets_dir,
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

    if available_files != expected_files:
        logger.info("Removing obsolete files from zip archive...")
        retain_archive_files(zip_path, expected_files)


if __name__ == "__main__":
    main()
