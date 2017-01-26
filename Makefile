run: build
	./shi

build: *.go
	go build -o shi ./cmd/shi/main.go

shi0:
	go build -o shi0 ./cmd/shi0/main.go
