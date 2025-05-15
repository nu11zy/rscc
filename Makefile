BIN_DIR=$(PWD)/bin

release: ## Build release binaries
	@echo "Building release binaries"
	@mkdir -p ${BIN_DIR}
	@cd pkg/agent && zip -q -r ${BIN_DIR}/agent.zip .
	@go build -o ${BIN_DIR}/rscc cmd/rscc/main.go

gen-ent: ## Generate ent models
	@echo "Generate ent models"
	@go generate $(PWD)/internal/database/ent

agent-vendor: ## Update vendor for agent
	@echo "Updating vendor for agent"
	@cd pkg/agent && go mod tidy && go mod vendor

clean:
	@rm -rf ${BIN_DIR}
	@rm rscc.db
	@rm -rf agents/

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
