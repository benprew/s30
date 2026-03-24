default:
	go run . -v mtg,duel

test:
	go test -count=10 ./...

winbuild:
	GOOS=windows GOARCH=amd64 go build -o s30.exe

winarmbuild:
	GOOS=windows GOARCH=arm64 go build -o s30_arm64.exe

macbuild:
	GOOS=darwin GOARCH=amd64 go build -o s30_mac

macarmbuild:
	GOOS=darwin GOARCH=arm64 go build -o s30_mac_arm

# https://ebitengine.org/en/documents/webassembly.html
webbuild:
	GOOS=js GOARCH=wasm go build -o s30.wasm github.com/benprew/s30
	scp s30.wasm /usr/local/go/lib/wasm/wasm_exec.js index.html main.html throwingbones@throwingbones.com:/var/www/html/throwingbones/ben/s30/

androidbuild:
	go run github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile bind -target android -javapkg com.benprew.s30 -o ./mobile/android/s30/s30.aar ./mobile

builddeps:
	sudo apt-get install libx11-dev libxrandr-dev libxinerama-dev libxcursor-dev libxi-dev libgl1-mesa-dev libxxf86vm-dev

fedorabuilddeps:
	sudo dnf install -y libX11-devel libXrandr-devel libXinerama-devel libXcursor-devel libXi-devel libXxf86vm-devel mesa-libGL-devel android-tools alsa-lib-devel

devdeps:
	sudo apt install imagemagick pngquant
