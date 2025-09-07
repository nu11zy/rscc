GIT_TAG=$(shell git describe --tags --abbrev=0)
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags="-s -w -extldflags '-static' -X github.com/nu11zy/rscc/internal/common/version.gitTag=$(GIT_TAG) -X github.com/nu11zy/rscc/internal/common/version.gitCommit=$(GIT_COMMIT) -X github.com/nu11zy/rscc/internal/common/version.gitBranch=$(GIT_BRANCH) -X github.com/nu11zy/rscc/internal/common/version.buildDate=$(BUILD_TIME)"
BIN_DIR=$(PWD)/bin

build: ## Build binary
	@echo "Building binary"
	@mkdir -p ${BIN_DIR}
	@cd pkg/agent && zip -q -r ${BIN_DIR}/agent.zip .
	@CGO_ENABLED=0 go build -trimpath ${LDFLAGS} -o ${BIN_DIR}/rscc cmd/rscc/main.go

build-all: ## Build binaries for platforms
	@echo "Building release binaries"
	@mkdir -p ${BIN_DIR}
	@cd pkg/agent && zip -q -r ${BIN_DIR}/agent.zip .
	@echo "Build for linux/amd64"
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath ${LDFLAGS} -o ${BIN_DIR}/rscc.linux.amd64 cmd/rscc/main.go
	@echo "Build for linux/arm64"
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath ${LDFLAGS} -o ${BIN_DIR}/rscc.linux.arm64 cmd/rscc/main.go
	@echo "Build for darwin/amd64"
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath ${LDFLAGS} -o ${BIN_DIR}/rscc.darwin.amd64 cmd/rscc/main.go
	@echo "Build for darwin/arm64"

gen-ent: ## Generate ent models
	@echo "Generate ent models"
	@go generate $(PWD)/internal/database/ent

agent-vendor: ## Update vendor for agent
	@echo "Updating vendor for agent"
	@cd pkg/agent && go mod tidy && go mod vendor

clean: ## Clean up
	@rm -rf ${BIN_DIR}
	@rm -rf agents/
	@rm rscc.db

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
