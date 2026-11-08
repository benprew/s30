# Card Lists

List of all cards in MTG, from Scryfall's bulk card list.

See: https://scryfall.com/docs/api/bulk-data

## Building Stripped Version

Run `update_cards_json.py` in `utils/`:

```bash
uv run python utils/update_cards_json.py <URL_OR_FILE>
```

Example:

```bash
uv run python utils/update_cards_json.py "https://data.scryfall.io/default-cards/default-cards-20260825210531.jsonl.gz"
```

### Options:
- `--sets`: Comma-separated list of set codes (default: `leb,arn,past,atq,phpr,4ed,drk`)
- `--art-sets`: Additional sets whose printings supply alternate artwork for existing playable cards (e.g. `--art-sets lea,2ed,unh`). This does not expand the playable card pool.
- `--exclude-name`: Card name to exclude globally (e.g. `--exclude-name "Chaos Orb"`)
- `--exclude-version`: Specific card version to exclude in format `Name:SetID` or `Name:SetID:CollectorNo` (e.g. `--exclude-version "El-Hajjâj:4ed"`)
- `--output-json`: Output path for uncompressed JSON (default: `assets/card_info/scryfall_cards.json`)
- `--output-zst`: Output path for zstd-compressed JSON (default: `assets/card_info/scryfall_cards.json.zst`)


## Alternate artwork

The updater keeps one rules definition per card name. Other printings from
`--sets` and `--art-sets` are stored in that card's `Artworks` list, with
`SetID`, `CollectorNo`, `SetName`, `Artist`, `BorderCropURL`, `PngURL`, and `ArtURL`.
The default printing is the first entry in the same list. Card definition IDs
are the exact card names; only artwork IDs include printing information.
Each variant has a stable ID based on its set, collector number, and card name.

```bash
uv run python utils/update_cards_json.py bulk.json --art-sets lea,2ed,unh
uv run python utils/download_card_images.py assets/card_info/scryfall_cards.json.zst
```

The downloader includes every artwork variant in `assets/art/cardimages.zip`.
Rebuild the game to use the regenerated `.zst` card database. Standalone builds
use `-tags embedded_card_images` to include the image archive; other builds
fetch images from the same URLs at runtime.

The current snapshot contains 642 card definitions and 1,010 artwork printings
from the configured sets. Regenerate artwork from the complete Scryfall
printing data. Reformatting an older database with one printing per name
cannot restore the discarded images. The Antiquities Mishra's Factory entry
includes collector numbers `80a`, `80b`, `80c`, and `80d`.
