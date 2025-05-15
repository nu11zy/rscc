BIN_DIR=$(PWD)/bin
LDFLAGS=-ldflags="-s -w"

build: ## Build release binaries
	@echo "Building release binaries..."
	@mkdir -p ${BIN_DIR}
	@cd pkg/agent && zip -q -r ${BIN_DIR}/agent.zip .
	@CGO_ENABLED=0 go build -trimpath ${LDFLAGS} -o ${BIN_DIR}/rscc cmd/rscc/main.go
	@rm -rf ${BIN_DIR}/agent.zip

gen-ent: ## Generate ent models
	@echo "Generate ent models"
	@go generate $(PWD)/internal/database/ent

agent-vendor: ## Update vendor for agent
	@echo "Updating vendor for agent"
	@cd pkg/agent && go mod tidy && go mod vendor

clean: ## Clean up
	@rm -rf ${BIN_DIR}
	@rm rscc.db
	@rm -rf agents/

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
