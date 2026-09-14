# iOS releases

The iOS application packages Ebitengine into a native iOS app using `ebitenmobile bind`.
The generated `Mobile.xcframework` contains the Go runtime, game logic, and an
Ebitengine view controller (`MobileEbitenViewController`). The native wrapper in `s30`
configures lifecycle hooks, screen orientation, and game saves in the application's
documents directory.

## Local Builds

To build the iOS project locally on macOS with Xcode installed:

1. Download and process card assets:

```bash
uv run python utils/download_card_images.py assets/card_info/scryfall_cards.json.zst
```

2. Build the Ebitengine xcframework:

```bash
go run github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@v2.9.9 bind \
  -target ios \
  -tags embedded_card_images \
  -o ./mobile/ios/s30/Mobile.xcframework \
  ./mobile
```

3. Open `mobile/ios/s30.xcodeproj` in Xcode or build via command line:

```bash
xcodebuild -project mobile/ios/s30.xcodeproj \
  -scheme s30 \
  -configuration Release \
  -destination 'generic/platform=iOS' \
  CODE_SIGNING_ALLOWED=NO
```

## GitHub Actions

The `.github/workflows/ios.yml` workflow builds `Mobile.xcframework` and creates
an iOS archive and `.ipa` package artifact. Release tags and manual workflow dispatches
can provide `version_name` which sets `MARKETING_VERSION`.
