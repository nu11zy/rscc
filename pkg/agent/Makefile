# Build variables
OS ?= darwin
ARCH ?= arm64
LDFLAGS ?= -ldflags="-s -w"
TAGS ?= -tags=debug
BUILD_MODE ?= 
OUTPUT_PATH ?= "/tmp/agent-debug"
SERVERS ?= 127.0.0.1:8080
PRIV_KEY ?= private_key_for_debug
FINGERPRINT ?= fingerprint_for_debug
SSH_CLIENT ?= SSH-2.0-OpenSSH_9.9

build: ## Build binary
	@echo "[*] Building binary"
	PRIV_KEY=${PRIV_KEY} SERVERS=${SERVERS} FINGERPRINT=${FINGERPRINT} SSH_CLIENT=${SSH_CLIENT} go run ./cmd/generate/main.go
	GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 go build -o ${OUTPUT_PATH} -trimpath -mod=vendor ${LDFLAGS} ${TAGS} ${BUILD_MODE} ./cmd/agent/main.go

garble: ## Build binary with garble
	@echo "[*] Building binary with garble"
	PRIV_KEY=${PRIV_KEY} SERVERS=${SERVERS} FINGERPRINT=${FINGERPRINT} SSH_CLIENT=${SSH_CLIENT} go run ./cmd/generate/main.go
	GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 garble -seed=random -tiny -literals build -o ${OUTPUT_PATH} -mod=vendor -trimpath ${LDFLAGS} ${TAGS} ${BUILD_MODE} ./cmd/agent/main.go
