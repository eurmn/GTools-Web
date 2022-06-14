all: build

build:
	rm -rf dist/
	go build -o dist/gtools.exe main.go
	mkdir -p dist/web/dist/
	cp -r web/dist/ dist/web/
	zip -r dist/gtools.zip dist/
