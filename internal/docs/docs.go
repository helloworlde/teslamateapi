// Package docs serves the API documentation surface (Scalar UI, Swagger
// JSON, and supporting static assets).
package docs

import (
	"embed"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
	"github.com/tobiasehlert/teslamateapi/internal/config"
	"github.com/watchakorn-18k/scalar-go"
	scalargin "github.com/watchakorn-18k/scalar-go/middleware/gin"
)

const scalarLocalScriptPath = "/api/docs/assets/scalar-api-reference.js"

//go:embed docs_assets/*
var docsAssetFS embed.FS

//go:embed generated/swagger.json
var embeddedSwaggerJSON []byte

// RegisterRoutes wires the docs routes onto the supplied /api router group.
func RegisterRoutes(api *gin.RouterGroup) {
	spec := swaggerSpecForDocs()
	api.GET("/docs", redirect)
	api.GET("/docs/", redirect)
	api.GET("/docs/swagger.json", openAPISpec)
	api.GET("/docs/assets/*filepath", serveAsset)
	api.GET("/docs/scalar", scalargin.Handler(&scalar.Options{
		CDN:         scalarLocalScriptPath,
		SpecContent: spec,
		CustomOptions: scalar.CustomOptions{
			PageTitle: "TeslaMateApi Reference",
		},
	}))
}

// redirect 将 /api/docs 与 /api/docs/ 永久跳转到 Scalar 文档页。
//
// @Summary  文档跳转
// @Description 将 /api/docs 与 /api/docs/ 永久重定向到 Scalar 渲染页。
// @Tags     system
// @Produce  json
// @Success  301 {string} string "已重定向"
// @Router   /docs [get]
func redirect(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/api/docs/scalar")
}

func swaggerSpecForDocs() map[string]interface{} {
	var spec map[string]interface{}
	if err := json.Unmarshal(embeddedSwaggerJSON, &spec); err != nil {
		panic("invalid embedded generated/swagger.json: " + err.Error())
	}
	if info, ok := spec["info"].(map[string]interface{}); ok {
		info["version"] = config.APIVersion
	}
	return spec
}

// serveAsset 返回 docs_assets 目录下的静态资源。
//
// @Summary  文档静态资源
// @Description 返回 Scalar 渲染所需的静态资源（JS、字体等）。
// @Tags     system
// @Produce  application/javascript
// @Produce  font/woff2
// @Produce  application/octet-stream
// @Param    filepath path string true "资源相对路径"
// @Success  200 {file} binary "资源二进制流"
// @Failure  400 {object} apicommon.APIErrorResponse "路径无效"
// @Failure  404 {object} apicommon.APIErrorResponse "资源不存在"
// @Router   /docs/assets/{filepath} [get]
func serveAsset(c *gin.Context) {
	requestPath := strings.TrimPrefix(c.Param("filepath"), "/")
	cleanPath := filepath.Clean(requestPath)
	if cleanPath == "." || strings.HasPrefix(cleanPath, "..") {
		apicommon.V2Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid docs asset path.", nil)
		return
	}

	content, err := docsAssetFS.ReadFile(filepath.ToSlash(filepath.Join("docs_assets", cleanPath)))
	if err != nil {
		apicommon.V2Error(c, http.StatusNotFound, "ASSET_NOT_FOUND", "Docs asset was not found.", nil)
		return
	}

	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Data(http.StatusOK, contentType(cleanPath), content)
}

func contentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".woff2":
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}

// openAPISpec 返回内嵌的 Swagger JSON 规范。
//
// @Summary  OpenAPI 规范
// @Description 返回内嵌的 Swagger 2.0 JSON 规范，可直接被 Scalar/Swagger UI 加载。
// @Tags     system
// @Produce  json
// @Success  200 {object} object "Swagger JSON 规范"
// @Router   /docs/swagger.json [get]
func openAPISpec(c *gin.Context) {
	c.JSON(http.StatusOK, swaggerSpecForDocs())
}
