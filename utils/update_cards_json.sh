#!/bin/bash





set -euo pipefail  # Exit on error, undefined vars, and pipe failures

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ASSETS_DIR="$PROJECT_ROOT/assets/card_info"

# Oracle Cards file info
ORACLE_CARDS_FILE=$1
ORACLE_CARDS_URL="https://data.scryfall.io/oracle-cards/$ORACLE_CARDS_FILE"

# Output files
OUTPUT_JSON="$ASSETS_DIR/scryfall_cards.json"
OUTPUT_ZST="$ASSETS_DIR/scryfall_cards.json.zst"
TEMP_DOWNLOAD="/tmp/$ORACLE_CARDS_FILE"

echo "Script directory: $SCRIPT_DIR"
echo "Project root: $PROJECT_ROOT"
echo "Assets directory: $ASSETS_DIR"

# Ensure assets directory exists
mkdir -p "$ASSETS_DIR"


# Download the Oracle Cards file to temp location
echo "Downloading $ORACLE_CARDS_URL..."
if ! wget -O "$TEMP_DOWNLOAD" "$ORACLE_CARDS_URL"; then
    echo "Error: Failed to download Oracle Cards file"
    exit 1
fi

# Process with jq filter
echo "Processing cards with jq filter..."
if ! jq 'map({
  ScryfallID: .id,
  OracleID: .oracle_id,
  CardName: .name,
  ManaCost: .mana_cost,
  Colors: .colors,
  ColorIdentity: .color_identity,
  Keywords: .keywords,
  CardType: (.type_line | split(" — ")[0] | split(" ")[0]),
  TypeLine: .type_line,
  Subtypes: (if (.type_line | contains(" — ")) then (.type_line | split(" — ")[1] | split(" ")) else [] end),
  Text: .oracle_text,
  Power: (if .power then (.power ) else null end),
  Toughness: (if .toughness then (.toughness ) else null end),
  SetName: .set_name,
  SetID: .set,
  Rarity: .rarity,
  Frame: .frame,
  FlavorText: .flavor_text,
  FrameEffects: .frame_effects,
  Watermark: .watermark,
  Artist: .artist,
  ManaProduction: .produced_mana
})' < "$TEMP_DOWNLOAD" > "$OUTPUT_JSON"; then
    echo "Error: Failed to process cards with jq"
    rm -f "$TEMP_DOWNLOAD"
    exit 1
fi

# Compress with zstd
echo "Compressing to zstd format..."
if ! zstd --rm -f "$OUTPUT_JSON" -o "$OUTPUT_ZST"; then
    echo "Error: Failed to compress with zstd"
    rm -f "$TEMP_DOWNLOAD"
    exit 1
fi

# Clean up temp file
rm -f "$TEMP_DOWNLOAD"

echo "Successfully created $OUTPUT_ZST"
echo "File size: $(du -h "$OUTPUT_ZST" | cut -f1)"
