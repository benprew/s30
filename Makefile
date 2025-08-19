default:
	go run .

test:
	go test ./...

winbuild:
	GOOS=windows GOARCH=amd64 go build -o s30.exe

macbuild:
	GOOS=darwin GOARCH=amd64 go build -o s30_mac

macarmbuild:
	GOOS=darwin GOARCH=arm64 go build -o s30_mac_arm

# https://ebitengine.org/en/documents/webassembly.html
webbuild:
	GOOS=js GOARCH=wasm go build -o s30.wasm github.com/benprew/s30
	scp s30.wasm /usr/local/go/lib/wasm/wasm_exec.js index.html main.html teamvite.com:/var/www/html/throwingbones/ben/s30/

builddeps:
	sudo apt-get install libx11-dev libxrandr-dev libxinerama-dev libxcursor-dev libxi-dev libgl1-mesa-dev libxxf86vm-dev

devdeps:
	sudo apt install imagemagick pngquant
