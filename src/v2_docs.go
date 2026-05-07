package main

import (
	"embed"
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

func RegisterDocsRoutes(api *gin.RouterGroup) {
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
		SpecContent: map[string]interface{}(buildOpenAPISpec()),
		CustomOptions: scalar.CustomOptions{
			PageTitle: "TeslaMateApi Reference",
		},
	}))
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
	c.JSON(http.StatusOK, buildOpenAPISpec())
}

func buildOpenAPISpec() gin.H {
	errorSchema := gin.H{
		"type": "object",
		"properties": gin.H{
			"error": gin.H{
				"type": "object",
				"properties": gin.H{
					"code":    gin.H{"type": "string"},
					"message": gin.H{"type": "string"},
					"details": gin.H{},
				},
				"required": []string{"code", "message"},
			},
		},
		"required": []string{"error"},
	}

	return gin.H{
		"openapi": "3.0.3",
		"info": gin.H{
			"title":       "TeslaMateApi",
			"description": "REST API for TeslaMate data, including V1 basic resources and V2 objective analytics.",
			"version":     apiVersion,
		},
		"servers": []gin.H{{"url": "/api"}},
		"tags": []gin.H{
			{"name": "V1", "description": "Existing V1 TeslaMate resource endpoints."},
			{"name": "V2 Summary", "description": "V2 analytics base and summary endpoints."},
		},
		"paths": gin.H{
			"/v1/":                                gin.H{"get": simpleOperation("V1", "V1 API root", "Returns the V1 API root status.")},
			"/v1/cars":                            gin.H{"get": simpleOperation("V1", "List cars", "Returns cars from TeslaMate.")},
			"/v1/cars/{CarID}":                    gin.H{"get": simpleCarOperation("V1", "Get car", "Returns one TeslaMate car.")},
			"/v1/cars/{CarID}/battery-health":     gin.H{"get": simpleCarOperation("V1", "Get battery health", "Returns V1 battery health data.")},
			"/v1/cars/{CarID}/charges":            gin.H{"get": simpleCarOperation("V1", "List charges", "Returns V1 charging sessions.")},
			"/v1/cars/{CarID}/charges/current":    gin.H{"get": simpleCarOperation("V1", "Get current charge", "Returns the active V1 charging session.")},
			"/v1/cars/{CarID}/charges/{ChargeID}": gin.H{"get": operationWithParams("V1", "Get charge", "Returns one V1 charging session.", []gin.H{carIDParam(), pathParam("ChargeID", "integer")})},
			"/v1/cars/{CarID}/command":            gin.H{"get": simpleCarOperation("V1", "List commands", "Returns enabled command information.")},
			"/v1/cars/{CarID}/command/{Command}":  gin.H{"post": operationWithParams("V1", "Run command", "Runs a V1 Tesla command when command support is enabled.", []gin.H{carIDParam(), pathParam("Command", "string")})},
			"/v1/cars/{CarID}/drives":             gin.H{"get": simpleCarOperation("V1", "List drives", "Returns V1 drives.")},
			"/v1/cars/{CarID}/drives/{DriveID}":   gin.H{"get": operationWithParams("V1", "Get drive", "Returns one V1 drive.", []gin.H{carIDParam(), pathParam("DriveID", "integer")})},
			"/v1/cars/{CarID}/logging":            gin.H{"get": simpleCarOperation("V1", "Get logging commands", "Returns V1 logging command information.")},
			"/v1/cars/{CarID}/logging/{Command}":  gin.H{"put": operationWithParams("V1", "Run logging command", "Runs a V1 logging command.", []gin.H{carIDParam(), pathParam("Command", "string")})},
			"/v1/cars/{CarID}/status":             gin.H{"get": simpleCarOperation("V1", "Get car status", "Returns V1 MQTT status.")},
			"/v1/cars/{CarID}/updates":            gin.H{"get": simpleCarOperation("V1", "List updates", "Returns V1 update history.")},
			"/v1/cars/{CarID}/wake_up":            gin.H{"post": simpleCarOperation("V1", "Wake car", "Runs the V1 wake_up command.")},
			"/v1/globalsettings":                  gin.H{"get": simpleOperation("V1", "Get global settings", "Returns TeslaMate settings.")},
			"/v2": gin.H{
				"get": gin.H{
					"tags":        []string{"V2 Summary"},
					"summary":     "V2 API capability information",
					"description": "Returns the V2 analytics API version, scope, and advertised feature groups.",
					"responses":   okJSONResponse("V2 API information"),
				},
			},
			"/v2/cars/{CarID}/analytics/summary": gin.H{
				"get": gin.H{
					"tags":        []string{"V2 Summary"},
					"summary":     "V2 period summary analytics",
					"description": "Returns objective driving, charging, parking, battery, update, and charging-cost metrics for one car in a selected period.",
					"parameters":  append([]gin.H{carIDParam()}, analyticsQueryParams()...),
					"responses": gin.H{
						"200": gin.H{"description": "V2 summary response", "content": jsonContent(gin.H{"type": "object"})},
						"400": gin.H{"description": "Invalid request", "content": jsonContent(errorSchema)},
						"404": gin.H{"description": "Car not found", "content": jsonContent(errorSchema)},
						"500": gin.H{"description": "Internal error", "content": jsonContent(errorSchema)},
					},
				},
			},
		},
	}
}

func simpleOperation(tag string, summary string, description string) gin.H {
	return operationWithParams(tag, summary, description, nil)
}

func simpleCarOperation(tag string, summary string, description string) gin.H {
	return operationWithParams(tag, summary, description, []gin.H{carIDParam()})
}

func operationWithParams(tag string, summary string, description string, params []gin.H) gin.H {
	operation := gin.H{
		"tags":        []string{tag},
		"summary":     summary,
		"description": description,
		"responses":   okJSONResponse("Successful response"),
	}
	if len(params) > 0 {
		operation["parameters"] = params
	}
	return operation
}

func okJSONResponse(description string) gin.H {
	return gin.H{"200": gin.H{"description": description, "content": jsonContent(gin.H{"type": "object"})}}
}

func jsonContent(schema gin.H) gin.H {
	return gin.H{"application/json": gin.H{"schema": schema}}
}

func carIDParam() gin.H {
	return pathParam("CarID", "integer")
}

func pathParam(name string, schemaType string) gin.H {
	return gin.H{
		"name":     name,
		"in":       "path",
		"required": true,
		"schema":   gin.H{"type": schemaType},
	}
}

func analyticsQueryParams() []gin.H {
	return []gin.H{
		queryParam("period", "string", []string{"day", "week", "month", "quarter", "year", "custom", "lifetime"}),
		queryParam("start", "string", nil),
		queryParam("end", "string", nil),
		queryParam("timezone", "string", nil),
		queryParam("compare", "string", []string{"none", "previous_period", "previous_year", "lifetime_average"}),
	}
}

func queryParam(name string, schemaType string, enum []string) gin.H {
	schema := gin.H{"type": schemaType}
	if len(enum) > 0 {
		schema["enum"] = enum
	}
	return gin.H{
		"name":     name,
		"in":       "query",
		"required": false,
		"schema":   schema,
	}
}
