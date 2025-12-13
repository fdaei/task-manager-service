BINARY := taskservice
CMD_PKG := ./cmd/taskservice
GOCACHE ?= $(PWD)/.gocache
COVER_PKGS ?= ./internal/... ./pkg/...

.PHONY: all build run test cover tidy
.PHONY: docker-build docker-up docker-up-observability docker-up-all docker-down docker-logs
.PHONY: air
.PHONY: test-html

all: build

build:
	@mkdir -p bin
	go build -buildvcs=false -o bin/$(BINARY) $(CMD_PKG)

run:
	go run $(CMD_PKG) serve

air:
	@if ! command -v air >/dev/null 2>&1; then \
		printf "air is not installed. Install with: go install github.com/cosmtrek/air@latest\n"; \
		exit 1; \
	fi
	GOCACHE=$(PWD)/.gocache air

docker-build:
	docker build -f deploy/Dockerfile -t $(BINARY):local .

docker-up:
	docker compose -f deploy/docker-compose.yml --profile dev up --build

docker-up-observability:
	docker compose -f deploy/docker-compose.yml --profile dev --profile observability up --build

# Convenience target to bring up app + observability stack together.
docker-up-all: docker-up-observability

docker-up-prod:
	docker compose -f deploy/docker-compose.yml --profile prod up --build

docker-down:
	docker compose -f deploy/docker-compose.yml down

docker-logs:
	docker compose -f deploy/docker-compose.yml logs -f

test:
	GOCACHE=$(GOCACHE) go test ./...

cover:
	GOCACHE=$(GOCACHE) go test -coverprofile=coverage.out $(COVER_PKGS)
	GOCACHE=$(GOCACHE) go tool cover -func=coverage.out

cover-packages:
	GOCACHE=$(GOCACHE) go test -cover $(COVER_PKGS)

tidy:
	go mod tidy

test-html:
	GOCACHE=$(GOCACHE) go run ./cmd/testreport -o test_report.html
