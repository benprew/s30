CARD_IMAGES := assets/art/cardimages.zip
CARD_DATA := assets/card_info/scryfall_cards.json.zst
DIST_DIR := dist
EMBEDDED_TAG := embedded_card_images

.PHONY: default run pprof dungeon test embeddedbuild cardimages winbuild macbuild webbuild builddeps fedorabuilddeps lint

default: build

run: cardimages
	go run -tags $(EMBEDDED_TAG) . -v mtg,duel

pprof: cardimages
	go run -tags $(EMBEDDED_TAG) . -pprof 127.0.0.1:6060 -v mtg,duel

dungeon:
	go run ./cmd/dungeon_test

test:
	go test -count=10 ./...

cardimages:
	uv run python utils/download_card_images.py $(CARD_DATA)
	test -f $(CARD_IMAGES)

build: cardimages
	mkdir -p $(DIST_DIR)
	go build -trimpath -tags $(EMBEDDED_TAG) -o $(DIST_DIR)/s30 .

winbuild: cardimages
	mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -trimpath -tags $(EMBEDDED_TAG) -o $(DIST_DIR)/s30.exe

macbuild: cardimages
	mkdir -p $(DIST_DIR)
	MACOSX_DEPLOYMENT_TARGET=12.0 CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -trimpath -tags $(EMBEDDED_TAG) -o $(DIST_DIR)/s30_mac_arm

# https://ebitengine.org/en/documents/webassembly.html
webbuild: cardimages
	mkdir -p $(DIST_DIR)
	GOOS=js GOARCH=wasm go build -trimpath -tags $(EMBEDDED_TAG) -o $(DIST_DIR)/s30.wasm github.com/benprew/s30
	scp $(DIST_DIR)/s30.wasm /usr/local/go/lib/wasm/wasm_exec.js index.html main.html throwingbones@throwingbones:/var/www/html/throwingbones/ben/s30/

builddeps:
	sudo apt-get install libx11-dev libxrandr-dev libxinerama-dev libxcursor-dev libxi-dev libgl1-mesa-dev libxxf86vm-dev
	pip3 install torch transformers scikit-learn scipy numpy

fedorabuilddeps:
	sudo dnf install -y libX11-devel libXrandr-devel libXinerama-devel libXcursor-devel libXi-devel libXxf86vm-devel mesa-libGL-devel android-tools alsa-lib-devel java-21-openjdk-devel

lint:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./...
	golangci-lint run --fix
	.venv/bin/ruff check --fix .
	.venv/bin/ruff format .
	.venv/bin/ty check .
