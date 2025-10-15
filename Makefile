# Environment variables for running the application.
# Override these with your own values, e.g. make run DOCKERHUB_USER=myuser
DOCKERHUB_USER?=your_dockerhub_user
DOCKERHUB_PASSWORD?=your_dockerhub_password
REPO_PATH?=your/repo
TARGET_PREFIX?=testnet-

RPC_URL?=http://localhost:1317
SOURCE_PREFIX?=release-
POLL_INTERVAL?=1m

HTTP_MAX_IDLE_CONNS?=100
HTTP_MAX_IDLE_CONNS_PER_HOST?=10
HTTP_MAX_CONNS_PER_HOST?=10

.PHONY: run
run:
	@DOCKERHUB_USER=$(DOCKERHUB_USER) \
	DOCKERHUB_PASSWORD=$(DOCKERHUB_PASSWORD) \
	REPO_PATH=$(REPO_PATH) \
	TARGET_PREFIX=$(TARGET_PREFIX) \
	RPC_URL=$(RPC_URL) \
	SOURCE_PREFIX=$(SOURCE_PREFIX) \
	POLL_INTERVAL=$(POLL_INTERVAL) \
	HTTP_MAX_IDLE_CONNS=$(HTTP_MAX_IDLE_CONNS) \
	HTTP_MAX_IDLE_CONNS_PER_HOST=$(HTTP_MAX_IDLE_CONNS_PER_HOST) \
	HTTP_MAX_CONNS_PER_HOST=$(HTTP_MAX_CONNS_PER_HOST) \
	@go run ./cmd/updater

.PHONY: build
build:
	@go build -o gopher-updater ./cmd/updater

.PHONY: test
test:
	@echo "--> Running tests..."
	@go test -v ./...

.PHONY: lint
lint:
	go mod tidy && git diff --exit-code
	go mod download
	go mod verify
	gofmt -s -w . && git diff --exit-code
	go vet ./...
	golangci-lint run

.PHONY: docker-build
docker-build: ## Build the docker image
	@docker build -t gopher-updater:latest .
