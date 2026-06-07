CORE_DIR=core
API_DIR=api

.PHONY: all core api cli setup docs
all: core api

cli:
	@echo "Building AP1 CLI..."
	cd cli && go build -o ../ap1-cli

core:
	@echo "Starting AP1 core..."
	cd $(CORE_DIR) && cargo run

api:
	@echo "Starting AP1 API server..."
	cd $(API_DIR) && go run main.go


docker:
	@echo "Starting AP1 services with Docker Compose..."
	docker compose -f docker/docker-compose.yml up --build

import-templates:
	@echo "Importing captive portal templates from ~/wifipumpkin3/config/templates to config/templates"
	@mkdir -p config/templates
	@cp -a $(HOME)/wifipumpkin3/config/templates/. config/templates/ || true

setup:
	@echo "Installing API dependencies..."
	cd $(API_DIR) && go mod tidy
	@echo "Checking Rust core..."
	cd $(CORE_DIR) && cargo check
	@echo "Building CLI..."
	cd cli && go build -o ../ap1-cli

docs:
	@echo "Open docs/setup.md for instructions"

.PHONY: cli
