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

# Ko build targets
.PHONY: ko-build
ko-build:
	ko build --local --bare .

.PHONY: ko-publish
ko-publish:
	ko build --bare .

.PHONY: ko-apply
ko-apply:
	ko apply -f config/

.PHONY: ko-multiarch
ko-multiarch:
	ko build --platform=linux/amd64,linux/arm64 --bare .

.PHONY: docker-build
docker-build:
	ko build --local --bare --platform=linux/amd64 .

# Linting and formatting
.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	go fmt ./...
	goimports -w .