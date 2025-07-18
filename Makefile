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
	# Note: AMD64 disabled to speed up builds - add back with --platform=linux/amd64,linux/arm64
	ko build --platform=linux/arm64 --bare .

.PHONY: docker-build
docker-build:
	ko build --local --bare --platform=linux/amd64 .

# Linting and formatting
.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	gofmt -w .
	goimports -w .

.PHONY: pre-commit
pre-commit: fmt test lint
	@echo "Pre-commit checks passed"

.PHONY: install-hooks
install-hooks:
	@command -v pre-commit >/dev/null 2>&1 || { echo "Installing pre-commit..."; pip install pre-commit; }
	pre-commit install
	@echo "Pre-commit hooks installed"