# TeslaMateApi 本地开发：生成 swag 产物、编译（校验 embed）等

.PHONY: help build test fmt vet docs swagger clean-docs check

GO ?= go
BIN_DIR ?= bin
BINARY ?= $(BIN_DIR)/teslamateapi

# 与 go.mod 中 swag 版本一致；无需全局安装 swag
SWAG_PKG := github.com/swaggo/swag/cmd/swag@v1.16.6
# 顶层 swagger 注解所在文件（@title / @version / @tag.* 等）
SWAG_GENERAL := internal/server/server.go
# swag 扫描目录（逗号分隔，generalInfo 文件必须在第一项）
SWAG_DIR := internal/server,internal/api/v1,internal/api/v2,internal/apicommon,internal/docs
# 与 internal/docs/docs.go //go:embed 一致；需提交 swagger.json 供 Docker 构建
DOCS_OUT ?= internal/docs/generated

help:
	@echo "用法: make <目标>"
	@echo "  docs / swagger  运行 swag init，生成 $(DOCS_OUT)/{swagger.json,swagger.yaml}"
	@echo "  build            编译 ./cmd/teslamateapi → $(BINARY)（校验 internal/docs 中 //go:embed docs_assets）"
	@echo "  test             go test ./internal/..."
	@echo "  fmt / vet        格式化、静态检查"
	@echo "  clean-docs      删除 swag 生成的文件（不删 $(DOCS_OUT) 下 .md 等）"
	@echo "  check            fmt + vet + test + build"
	@echo "说明: Scalar 使用 internal/docs/docs.go 嵌入的 generated/swagger.json（已去掉 definitions 的 main. 前缀）。"

docs swagger:
	$(GO) run $(SWAG_PKG) init -g $(SWAG_GENERAL) -d $(SWAG_DIR) -o $(DOCS_OUT) --parseInternal
	rm -f $(DOCS_OUT)/docs.go
	python3 scripts/normalize_swagger_main_prefix.py $(DOCS_OUT)

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) ./cmd/teslamateapi

test:
	$(GO) test ./internal/... -count=1

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./internal/... ./cmd/...

clean-docs:
	rm -f $(DOCS_OUT)/docs.go $(DOCS_OUT)/swagger.json $(DOCS_OUT)/swagger.yaml

check: fmt vet test build
