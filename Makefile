default:
	go run .

winbuild:
	GOOS=windows GOARCH=amd64 go build -o s30.exe
