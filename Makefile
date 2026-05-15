SHELL := /bin/bash
export BASH_ENV := $(HOME)/.bashrc

SOPS_KMS_ARN ?= arn:aws:kms:us-east-1:914788356809:alias/tilde-app-dev-sops
CATALOG_EMBEDDING_MODEL ?= text-embedding-3-small
CATALOG_EMBEDDING_DIMENSIONS ?= 768
DOCS_REPO_DIR ?= docs

.PHONY: test test-unit test-e2e test-provider test-provider-tool ensure-docs-submodule generate-metadata generate-docs generate-catalog generate-catalog-provider generate-catalog-tool build build-all sops-encrypt sops-decrypt env-secrets sops-encrypt-provider-test-secrets help

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-38s %s\n", $$1, $$2}'

test: test-unit ## Run default test suite

test-unit: generate-metadata ## Run internal unit tests
	go test ./internal/...

test-e2e: generate-metadata ## Run all provider e2e tests
	go test ./providers/... -run TestE2E

test-provider: generate-metadata ## Run e2e tests for one provider: make test-provider PROVIDER=google
	@test -n "$(PROVIDER)" || (echo "PROVIDER is required" && exit 2)
	go test ./providers/$(PROVIDER)/... -run TestE2E

test-provider-tool: generate-metadata ## Run e2e tests for one tool: make test-provider-tool PROVIDER=google TOOL=send-email
	@test -n "$(PROVIDER)" || (echo "PROVIDER is required" && exit 2)
	@test -n "$(TOOL)" || (echo "TOOL is required" && exit 2)
	go test ./providers/$(PROVIDER)/$(TOOL) -run TestE2E

ensure-docs-submodule: ## Ensure shared docs git submodule is initialized
	git submodule update --init --recursive docs

generate-metadata: ## Generate static Go metadata/schema files from provider YAML
	go run ./tools/generatemetadata

generate-docs: generate-metadata ensure-docs-submodule ## Generate Mintlify provider/tool docs from metadata
	go run ./tools/generatedocs --docs-root $(DOCS_REPO_DIR)

generate-catalog: generate-metadata ## Generate embedded catalogue files for all tools
	@set -a; [ -f .env ] && . ./.env; [ -f .env.secrets ] && . ./.env.secrets; set +a; go run ./tools/embedcatalog --model $(CATALOG_EMBEDDING_MODEL) --dimensions $(CATALOG_EMBEDDING_DIMENSIONS)

generate-catalog-provider: generate-metadata ## Generate embedded catalogue files for one provider: make generate-catalog-provider PROVIDER=google
	@test -n "$(PROVIDER)" || (echo "PROVIDER is required" && exit 2)
	@set -a; [ -f .env ] && . ./.env; [ -f .env.secrets ] && . ./.env.secrets; set +a; go run ./tools/embedcatalog --model $(CATALOG_EMBEDDING_MODEL) --dimensions $(CATALOG_EMBEDDING_DIMENSIONS) --provider $(PROVIDER)

generate-catalog-tool: generate-metadata ## Generate embedded catalogue files for one tool: make generate-catalog-tool PROVIDER=google TOOL=send-email
	@test -n "$(PROVIDER)" || (echo "PROVIDER is required" && exit 2)
	@test -n "$(TOOL)" || (echo "TOOL is required" && exit 2)
	@set -a; [ -f .env ] && . ./.env; [ -f .env.secrets ] && . ./.env.secrets; set +a; go run ./tools/embedcatalog --model $(CATALOG_EMBEDDING_MODEL) --dimensions $(CATALOG_EMBEDDING_DIMENSIONS) --provider $(PROVIDER) --tool $(TOOL)

build: generate-metadata ## Build local factory binary
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/factory ./cmd/factory

build-all: generate-metadata ## Build static binaries for Linux, macOS, and Windows
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/factory-linux-amd64 ./cmd/factory
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/factory-linux-arm64 ./cmd/factory
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/factory-darwin-amd64 ./cmd/factory
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/factory-darwin-arm64 ./cmd/factory
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/factory-windows-amd64.exe ./cmd/factory

sops-encrypt: ## Encrypt root secrets.yaml using SOPS + AWS KMS
	sops encrypt --kms $(SOPS_KMS_ARN) secrets.yaml > secrets.enc.yaml

sops-decrypt: ## Decrypt root secrets.enc.yaml to secrets.yaml, then generate .env.secrets
	@if [ ! -f secrets.yaml ]; then \
		sops decrypt secrets.enc.yaml > secrets.yaml; \
	else \
		echo "secrets.yaml already exists. Not decrypting to prevent overwriting."; \
	fi
	./scripts/load-secrets.sh

env-secrets: ## Decrypt local secrets.enc.yaml directly into .env.secrets
	@tmp="$$(mktemp)"; \
	trap 'rm -f "$$tmp"' EXIT; \
	sops decrypt secrets.enc.yaml > "$$tmp"; \
	./scripts/load-secrets.sh "$$tmp" .env.secrets

sops-encrypt-provider-test-secrets: ## Encrypt provider test secrets: make sops-encrypt-provider-test-secrets PROVIDER=google
	@test -n "$(PROVIDER)" || (echo "PROVIDER is required" && exit 2)
	sops encrypt --kms $(SOPS_KMS_ARN) providers/$(PROVIDER)/test_secrets.yaml > providers/$(PROVIDER)/test_secrets.enc.yaml
