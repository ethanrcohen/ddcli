BINARY := ddcli
PKG := github.com/ethan/ddcli

.PHONY: build test test-unit test-integration lint clean install

build:
	go build -o $(BINARY) .

install: build
	mv $(BINARY) $(GOPATH)/bin/$(BINARY)

test:
	go test ./... -v

test-unit:
	go test ./... -v -short

test-integration:
	go test ./... -v -tags integration -run Integration

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/
