DIST_DIR := dist
GOROOT   := $(shell go env GOROOT)

MKDIR_DIST      := mkdir -p $(DIST_DIR)
COPY_WEB_ASSETS := cp "$(GOROOT)/lib/wasm/wasm_exec.js" index.html main.html $(DIST_DIR)/
RM_DIST         := rm -rf $(DIST_DIR)
PYTHON          := python3
TEST_COMMAND    := go test

# OS detection: Windows sets OS=Windows_NT; macOS and Linux leave it unset.
# uname is only called on non-Windows so it is safe on all platforms.
ifeq ($(OS),Windows_NT)
# Windows — check whether the shell is cmd.exe or a POSIX shell (Git Bash / MSYS2).
	ifneq ($(filter cmd cmd.exe,$(notdir $(subst \,/,$(SHELL)))),)
		MKDIR_DIST       := if not exist $(DIST_DIR) mkdir $(DIST_DIR)
		COPY_WEB_ASSETS  := copy /y "$(subst /,\,$(GOROOT))\lib\wasm\wasm_exec.js" $(DIST_DIR)\ >nul && copy /y index.html $(DIST_DIR)\ >nul && copy /y main.html $(DIST_DIR)\ >nul
		RM_DIST          := if exist $(DIST_DIR) rmdir /s /q $(DIST_DIR)
		PYTHON           := python
	endif
else ifneq ($(shell uname -s 2>/dev/null),Darwin)
# Linux — tests require a virtual framebuffer.
	TEST_COMMAND := xvfb-run -a go test
endif

.PHONY: default run pprof duelprofile test test-flaky build winbuild macbuild webbuild webdeploy clean builddeps fedorabuilddeps osdeps fedoraosdeps pydeps lint

default: build

run:
	go run . -v mtg,duel

webrun: webbuild
	$(PYTHON) -m http.server -d $(DIST_DIR) 8080

pprof:
	go run -tags pprof . -pprof 127.0.0.1:6060 -v mtg,duel

duelprofile:
	$(MKDIR_DIST)
	go build -trimpath -o $(DIST_DIR)/duel_profile ./cmd/duel_test

test:
	$(TEST_COMMAND) ./...

test-flaky:
	$(TEST_COMMAND) -count=20 -shuffle=on

build:
	$(MKDIR_DIST)
	go build -trimpath -o $(DIST_DIR)/s30 .

winbuild: export GOOS=windows
winbuild: export GOARCH=amd64
winbuild:
	$(MKDIR_DIST)
	go build -trimpath -o $(DIST_DIR)/s30.exe .

macbuild: export MACOSX_DEPLOYMENT_TARGET=12.0
macbuild: export CGO_ENABLED=1
macbuild: export GOOS=darwin
macbuild: export GOARCH=arm64
macbuild:
	$(MKDIR_DIST)
	go build -trimpath -o $(DIST_DIR)/s30_mac_arm .

# https://ebitengine.org/en/documents/webassembly.html
webbuild: export GOOS=js
webbuild: export GOARCH=wasm
webbuild:
	$(MKDIR_DIST)
	go build -trimpath -o $(DIST_DIR)/s30.wasm .
	$(COPY_WEB_ASSETS)

webdeploy: webbuild
	scp $(DIST_DIR)/s30.wasm $(DIST_DIR)/wasm_exec.js $(DIST_DIR)/index.html $(DIST_DIR)/main.html throwingbones@throwingbones:/var/www/html/throwingbones/ben/s30/

clean:
	$(RM_DIST)

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
	uv run ruff check --fix .
	uv run ruff format .
	uv run ty check .
