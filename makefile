all: build

build:
	rm -rf dist/
	go build -o dist/gtools.exe main.go
	cd web && pnpm build
	mkdir -p dist/web/dist/
	cp -r web/dist/ dist/web/
	zip -qr dist/gtools.zip dist/

backend:
	rm -f dist/gtools.exe
	go build -o dist/gtools.exe main.go

debug:
	go run main.go --debug & cd web && pnpm dev