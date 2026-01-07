#!/bin/bash

set -euo pipefail  # Exit on error, undefined vars, and pipe failures

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ASSETS_DIR="$PROJECT_ROOT/assets/card_info"

# Oracle Cards file info
#
# Use default cards set from Scryfall bulk data API
# https://data.scryfall.io/default-cards/default-cards-20250807213540.json
#
# Details about the bulk data api:
# https://scryfall.com/docs/api/bulk-data
INPUT_FILE=$1

# Output files
OUTPUT_JSON="$ASSETS_DIR/scryfall_cards.json"
OUTPUT_ZST="$ASSETS_DIR/scryfall_cards.json.zst"

echo "Script directory: $SCRIPT_DIR"
echo "Project root: $PROJECT_ROOT"
echo "Assets directory: $ASSETS_DIR"

# Ensure assets directory exists
mkdir -p "$ASSETS_DIR"

# The "Old school" block
# lea - alpha
# 2ed - unlimited
# arn - arabian nights
# leg - legends
# atq - antiquities
# drk - the dark
# fem -fallen empires
# phpr - arena & sewars of estark
# past - Shandalar astral set

# Process with jq filter
# Note: past is the Astral Cards Set
echo "Processing cards with jq filter..."
if ! jq 'map(select((.set == "lea" or .set == "2ed" or .set == "arn" or .set == "leg" or .set == "atq" or .set == "drk" or .set == "past" or .set == "fem" or .set == "phpr"))) | map({
  CardName: .name,
  ManaCost: .mana_cost,
  Colors: .colors,
  ColorIdentity: .color_identity,
  Keywords: .keywords,
  TypeLine: .type_line,
  Text: .oracle_text,
  Power: (if .power then (.power ) else null end),
  Toughness: (if .toughness then (.toughness ) else null end),
  SetName: .set_name,
  SetID: .set,
  CollectorNo: .collector_number,
  Rarity: .rarity,
  Frame: .frame,
  FlavorText: .flavor_text,
  FrameEffects: .frame_effects,
  Watermark: .watermark,
  Artist: .artist,
  ManaProduction: .produced_mana,
  PriceUSD: (if .prices.usd then (.prices.usd) else .prices.eur end),
  VintageRestricted: (if .legalities.vintage == "restricted" then true else false end),
  PngURL: .image_uris.png,
  ArtURL: .image_uris.art_crop
})' < "$INPUT_FILE" > "$OUTPUT_JSON"; then
    echo "Error: Failed to process cards with jq"
    exit 1
fi

# Compress with zstd
echo "Compressing to zstd format..."
if ! zstd  -f "$OUTPUT_JSON" -o "$OUTPUT_ZST"; then
    echo "Error: Failed to compress with zstd"
    exit 1
fi

echo "Successfully created $OUTPUT_ZST"
echo "File size: $(du -h "$OUTPUT_ZST" | cut -f1)"
