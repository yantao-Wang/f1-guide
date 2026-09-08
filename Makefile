# 「走近围场 · F1咨讯」开发工作流入口
# 用法：make help

.PHONY: help build test cover lint run tools migrate-up migrate-down seed docker-up docker-down clean

help: ## 列出全部命令
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## 编译全部包
	go build ./...

test: ## 运行全部测试
	go test ./...

cover: ## 测试 + 覆盖率报告（口径：internal + pkg，与 CI 一致）
	go test -coverprofile=cover.out ./internal/... ./pkg/...
	go tool cover -func=cover.out | tail -1

lint: ## 静态检查（golangci-lint，CI 同款）
	golangci-lint run

run: ## 本地运行 API（默认 :8080）
	go run ./cmd/api

tools: ## 安装开发工具（golangci-lint）
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

migrate-up: ## 应用数据库迁移
	go run ./cmd/migrate up

migrate-down: ## 回滚最近一条迁移
	go run ./cmd/migrate down

seed: ## 导入本地开发示例数据（占位内容，正式内容走内容生产流水线）
	docker compose exec -T db psql -U f1guide -d f1guide < seeds/dev.sql

docker-up: ## 启动本地 PostgreSQL
	docker compose up -d db

docker-down: ## 停止并删除本地容器
	docker compose down

clean: ## 清理构建产物
	rm -rf bin/ cover.out
