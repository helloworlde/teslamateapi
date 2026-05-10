package main

import (
	"embed"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/watchakorn-18k/scalar-go"
	scalargin "github.com/watchakorn-18k/scalar-go/middleware/gin"
)

const scalarLocalScriptPath = "/api/docs/assets/scalar-api-reference.js"

//go:embed docs_assets/*
var docsAssetFS embed.FS

//go:embed generated/swagger.json
var embeddedSwaggerJSON []byte

func RegisterDocsRoutes(api *gin.RouterGroup) {
	spec := swaggerSpecForDocs()
	api.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/api/docs/scalar")
	})
	api.GET("/docs/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/api/docs/scalar")
	})
	api.GET("/docs/swagger.json", V2OpenAPISpec)
	api.GET("/docs/assets/*filepath", ServeDocsAsset)
	api.GET("/docs/scalar", scalargin.Handler(&scalar.Options{
		CDN:         scalarLocalScriptPath,
		SpecContent: spec,
		CustomOptions: scalar.CustomOptions{
			PageTitle: "TeslaMateApi Reference",
		},
	}))
}

func swaggerSpecForDocs() map[string]interface{} {
	var spec map[string]interface{}
	if err := json.Unmarshal(embeddedSwaggerJSON, &spec); err != nil {
		panic("invalid embedded generated/swagger.json: " + err.Error())
	}
	if info, ok := spec["info"].(map[string]interface{}); ok {
		info["version"] = apiVersion
	}
	return spec
}

func ServeDocsAsset(c *gin.Context) {
	requestPath := strings.TrimPrefix(c.Param("filepath"), "/")
	cleanPath := filepath.Clean(requestPath)
	if cleanPath == "." || strings.HasPrefix(cleanPath, "..") {
		v2Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid docs asset path.", nil)
		return
	}

	content, err := docsAssetFS.ReadFile(filepath.ToSlash(filepath.Join("docs_assets", cleanPath)))
	if err != nil {
		v2Error(c, http.StatusNotFound, "ASSET_NOT_FOUND", "Docs asset was not found.", nil)
		return
	}

	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Data(http.StatusOK, docsAssetContentType(cleanPath), content)
}

func docsAssetContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".woff2":
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}

func V2OpenAPISpec(c *gin.Context) {
	c.JSON(http.StatusOK, swaggerSpecForDocs())
}
