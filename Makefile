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
	ko build --local --preserve-import-paths .

.PHONY: ko-publish
ko-publish:
	ko build --preserve-import-paths .

.PHONY: ko-apply
ko-apply:
	ko apply -f config/

.PHONY: docker-build
docker-build:
	ko build --local --preserve-import-paths --platform=linux/amd64 .