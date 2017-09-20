
build:
	@go build -o client ./...

clean:
	@rm -f client

test:
	@go test -short ./...

lint:
	@go vet ./..

.PHONY: build test lint clean