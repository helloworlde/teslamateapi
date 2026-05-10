# TeslaMateApi 本地开发：生成 swag 产物、编译（校验 embed）等

.PHONY: help build test fmt vet docs swagger clean-docs check

GO ?= go
BIN_DIR ?= bin
BINARY ?= $(BIN_DIR)/teslamateapi

# 与 go.mod 中 swag 版本一致；无需全局安装 swag
SWAG_PKG := github.com/swaggo/swag/cmd/swag@v1.16.6
SWAG_GENERAL := webserver.go
SWAG_DIR := src
# 与 v2_docs.go //go:embed 一致；需提交 swagger.json 供 Docker（仅 COPY src）构建
DOCS_OUT ?= src/generated

help:
	@echo "用法: make <目标>"
	@echo "  docs / swagger  运行 swag init，生成 $(DOCS_OUT)/{docs.go,swagger.json,swagger.yaml}"
	@echo "  build            编译 ./src → $(BINARY)（校验 v2_docs.go 中 //go:embed docs_assets）"
	@echo "  test             go test ./src/..."
	@echo "  fmt / vet        格式化、静态检查"
	@echo "  clean-docs      删除 swag 生成的三个文件（不删 $(DOCS_OUT) 下 .md 等）"
	@echo "  check            fmt + vet + test + build"
	@echo "说明: 线上 Scalar 使用 src/v2_docs.go 内嵌的 OpenAPI 3，与 swag 产物独立。"

docs swagger:
	$(GO) run $(SWAG_PKG) init -g $(SWAG_GENERAL) -d $(SWAG_DIR) -o $(DOCS_OUT)
	rm -f $(DOCS_OUT)/docs.go

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) ./src/...

test:
	$(GO) test ./src/... -count=1

fmt:
	$(GO) fmt ./src/...

vet:
	$(GO) vet ./src/...

clean-docs:
	rm -f $(DOCS_OUT)/docs.go $(DOCS_OUT)/swagger.json $(DOCS_OUT)/swagger.yaml

check: fmt vet test build
