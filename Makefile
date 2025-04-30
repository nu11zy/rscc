release: ## Build release binaries
	@echo "Building release binaries..."
	@mkdir -p bin
	@cd pkg/agent && zip -q -r ../../bin/agent.zip .
	@go build -o bin/rscc cmd/rscc/main.go

gen-ent: ## Generate ent models
	@echo "Generate ent models..."
	@go generate $(PWD)/internal/database/ent

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
