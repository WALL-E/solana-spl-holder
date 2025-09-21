# Solana SPL Holder Makefile

# 变量定义
APP_NAME = solana-spl-holder
MAIN_FILE = server/main.go
BUILD_DIR = build

# 构建信息
BUILD_TIME = $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 链接标志
LDFLAGS = -ldflags="-s -w -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'"

# 默认目标
.PHONY: all
all: clean deps build

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# 构建应用
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Build completed: $(BUILD_DIR)/$(APP_NAME)"

# 构建并运行 (devnet)
.PHONY: run-dev
run-dev:
	@echo "Running in devnet mode..."
	cd server && go run main.go --rpc_url https://api.devnet.solana.com --interval_time 30

# 构建并运行 (localnet)
.PHONY: run-local
run-local:
	@echo "Running in localnet mode..."
	cd server && go run main.go --rpc_url http://localhost:8899 --interval_time 30

# 构建并运行 (mainnet)
.PHONY: run-mainnet
run-mainnet:
	@echo "Running in mainnet mode..."
	@if [ -z "$$SOLANA_RPC" ]; then \
		echo "Error: SOLANA_RPC environment variable is not set"; \
		exit 1; \
	fi
	cd server && go run main.go --rpc_url $$SOLANA_RPC --interval_time 300

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./test/...

# 运行测试并显示覆盖率
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -cover ./test/...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码检查
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

# 初始化数据库
.PHONY: init-db
init-db:
	@echo "Initializing database..."
	@echo "Please run the SQL script manually:"
	@echo "mysql -u root -p < setup/init_database.sql"

# 清理构建文件
.PHONY: clean
clean:
	@echo "Cleaning build files..."
	rm -rf $(BUILD_DIR)
	rm -f $(APP_NAME)

# 安装到系统
.PHONY: install
install: build
	@echo "Installing $(APP_NAME)..."
	cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/

# 卸载
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(APP_NAME)..."
	rm -f /usr/local/bin/$(APP_NAME)

# 显示帮助
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Clean, install deps and build"
	@echo "  deps         - Install dependencies"
	@echo "  build        - Build the application"
	@echo "  run-dev      - Run in devnet mode"
	@echo "  run-local    - Run in localnet mode"
	@echo "  run-mainnet  - Run in mainnet mode (requires SOLANA_RPC env var)"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  init-db      - Show database initialization instructions"
	@echo "  clean        - Clean build files"
	@echo "  install      - Install to /usr/local/bin"
	@echo "  uninstall    - Remove from /usr/local/bin"
	@echo "  help         - Show this help"

# 开发环境快速启动
.PHONY: dev
dev: deps fmt vet run-dev

# 生产环境构建
.PHONY: prod
prod: clean deps fmt vet test build
	@echo "Production build completed!"