APP := aijiaoxue-api
CONFIG ?= config.yaml
PKG := ./...

.PHONY: help run build seed test lint tidy db-init db-seed migrate migrate-down migrate-version verify

help:
	@echo "make run      启动开发服务（air 热重载，未安装则回退 go run）"
	@echo "make build    编译到 bin/$(APP)"
	@echo "make seed     用 bcrypt 覆写演示账号密码哈希（默认 123456）"
	@echo "make test     运行全部单测"
	@echo "make lint     go vet + gofmt 检查"
	@echo "make tidy     go mod tidy"
	@echo "make db-init  执行 database/schema.sql（建库建表，需 root 密码）"
	@echo "make db-seed  执行 database/seed.sql（清空并写入种子数据）"
	@echo "make migrate  执行 golang-migrate 升级到最新版本（V1 基线 + V2 起新表）"
	@echo "make migrate-down  回滚一个迁移版本"
	@echo "make migrate-version  查看当前迁移版本"
	@echo "make verify   端到端验收脚本 scripts/verify.sh"

run:
	@if command -v air >/dev/null 2>&1; then \
		air -c .air.toml; \
	else \
		echo "air 未安装（go install github.com/air-verse/air@latest），回退为 go run"; \
		go run ./cmd/server -config $(CONFIG); \
	fi

build:
	go build -o bin/$(APP) ./cmd/server

seed:
	go run ./cmd/seed -config $(CONFIG)

test:
	go test $(PKG) -count=1

lint:
	go vet $(PKG)
	@files=$$(gofmt -l .); \
	if [ -n "$$files" ]; then echo "以下文件未通过 gofmt:"; echo "$$files"; exit 1; fi; \
	echo "gofmt/vet 通过"

tidy:
	go mod tidy

db-init:
	mysql -u root -p < database/schema.sql

db-seed:
	mysql -u root -p aijiaoxue < database/seed.sql

migrate:
	go run ./cmd/migrate -config $(CONFIG) up

migrate-down:
	go run ./cmd/migrate -config $(CONFIG) down 1

migrate-version:
	go run ./cmd/migrate -config $(CONFIG) version

verify:
	bash scripts/verify.sh
