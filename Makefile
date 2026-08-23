DIST_DIR := dist
TEST_COUNT := 5

.PHONY: default run pprof duelprofile test winbuild macbuild webbuild webdeploy builddeps fedorabuilddeps osdeps fedoraosdeps pydeps lint

default: build

run:
	go run . -v mtg,duel

webrun: webbuild
	python3 -m http.server -d $(DIST_DIR) 8080

pprof:
	go run -tags pprof . -pprof 127.0.0.1:6060 -v mtg,duel

duelprofile:
	mkdir -p $(DIST_DIR)
	go build -trimpath -o $(DIST_DIR)/duel_profile ./cmd/duel_test

test:
ifeq ($(shell uname -s),Linux)
	xvfb-run -a go test -count=$(TEST_COUNT) ./...
else
	go test -count=$(TEST_COUNT) ./...
endif

build:
	mkdir -p $(DIST_DIR)
	go build -trimpath -o $(DIST_DIR)/s30 .

winbuild:
	mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -trimpath -o $(DIST_DIR)/s30.exe .

macbuild:
	mkdir -p $(DIST_DIR)
	MACOSX_DEPLOYMENT_TARGET=12.0 CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -trimpath -o $(DIST_DIR)/s30_mac_arm .

# https://ebitengine.org/en/documents/webassembly.html
webbuild:
	mkdir -p $(DIST_DIR)
	GOOS=js GOARCH=wasm go build -trimpath -o $(DIST_DIR)/s30.wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" index.html main.html $(DIST_DIR)/

webdeploy: webbuild
	scp $(DIST_DIR)/s30.wasm $(DIST_DIR)/wasm_exec.js $(DIST_DIR)/index.html $(DIST_DIR)/main.html throwingbones@throwingbones:/var/www/html/throwingbones/ben/s30/

APT_DEPS := libasound2-dev libx11-dev libxrandr-dev libxinerama-dev libxcursor-dev libxi-dev libgl1-mesa-dev libxxf86vm-dev xvfb
DNF_DEPS := libX11-devel libXrandr-devel libXinerama-devel libXcursor-devel libXi-devel libXxf86vm-devel mesa-libGL-devel alsa-lib-devel xorg-x11-server-Xvfb

osdeps:
	sudo apt-get install -y $(APT_DEPS)

fedoraosdeps:
	sudo dnf install -y $(DNF_DEPS)

pydeps:
	uv sync

builddeps: osdeps pydeps

fedorabuilddeps: fedoraosdeps pydeps

lint:
	modernize -fix ./...
	golangci-lint run --fix
	.venv/bin/ruff check --fix .
	.venv/bin/ruff format .
	.venv/bin/ty check .
