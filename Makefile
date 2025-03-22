default:
	go run .

winbuild:
	GOOS=windows GOARCH=amd64 go build -o s30.exe

macbuild:
	GOOS=darwin GOARCH=amd64 go build -o s30_mac

macarmbuild:
	GOOS=darwin GOARCH=arm64 go build -o s30_mac_arm

# https://ebitengine.org/en/documents/webassembly.html
webbuild:
	GOOS=js GOARCH=wasm go build -o s30.wasm github.com/benprew/s30
	cp /usr/local/go/lib/wasm/wasm_exec.js .
	scp  s30.wasm wasm_exec.js index.html main.html teamvite.com:/var/www/html/throwingbones/ben/s30/
