.PHONY: shi shi0

run: build
	./shi

build: shi

shi: *.go
	go build -o shi ./cmd/shi/main.go

link:
	mkdir -p $(GOPATH)/src/github.com/kiasaki
	ln -s $(realpath .) $(GOPATH)/src/github.com/kiasaki/shi
