.PHONY: test
test:
	go test ./... -v

.PHONY: test-short
test-short:
	go test ./... -short

.PHONY: run
run:
	go run main.go

.PHONY: build
build:
	go build -o xipe main.go

.PHONY: clean
clean:
	rm -f xipe
	go clean

.PHONY: deps
deps:
	go mod download
	go mod tidy