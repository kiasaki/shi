run: build
	./shi

build: *.go
	go build -o shi
