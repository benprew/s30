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
- `--sets`: Comma-separated list of set codes (default: `2ed,4ed,arn,atq,past,phpr`)
- `--exclude-name`: Card name to exclude globally (e.g. `--exclude-name "Chaos Orb"`)
- `--exclude-version`: Specific card version to exclude in format `Name:SetID` or `Name:SetID:CollectorNo` (e.g. `--exclude-version "El-Hajjâj:4ed"`)
- `--output-json`: Output path for uncompressed JSON (default: `assets/card_info/scryfall_cards.json`)
- `--output-zst`: Output path for zstd-compressed JSON (default: `assets/card_info/scryfall_cards.json.zst`)
