.PHONY: build run test lint clean docker-build docker-push deploy

# Variables
APP_NAME := gagos
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
DOCKER_REGISTRY := netstudioge
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(APP_NAME)

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
BINARY_DIR := bin

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags="-w -s -X main.version=$(VERSION)" -o $(BINARY_DIR)/$(APP_NAME) ./cmd/gagos

# Run the application locally
run:
	@echo "Running $(APP_NAME)..."
	$(GOCMD) run ./cmd/gagos

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_DIR)
	rm -rf .cache

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -f deploy/docker/Dockerfile .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

# Push Docker image
docker-push: docker-build
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# Deploy to Kubernetes
deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -k deploy/kubernetes/base

# Deploy to specific environment
deploy-dev:
	kubectl apply -k deploy/kubernetes/overlays/dev

deploy-staging:
	kubectl apply -k deploy/kubernetes/overlays/staging

deploy-prod:
	kubectl apply -k deploy/kubernetes/overlays/prod

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linter"
	@echo "  deps         - Download dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-push  - Push Docker image to registry"
	@echo "  deploy       - Deploy to Kubernetes (base)"
	@echo "  deploy-dev   - Deploy to dev environment"
	@echo "  deploy-staging - Deploy to staging environment"
	@echo "  deploy-prod  - Deploy to production environment"
