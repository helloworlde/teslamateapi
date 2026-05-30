package main

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
	scalar "github.com/watchakorn-18k/scalar-go"
)

// openapiSpec embeds the swag-generated OpenAPI spec at build time so the
// binary is self-contained — no extra files to ship alongside it. The path
// is relative to this source file; `swag init` writes the YAML next to its
// docs.go in src/docs/.
//
//go:embed docs/swagger.yaml
var openapiSpec []byte

// openapiYAML serves the raw spec at /api/openapi.yaml so external tooling
// (Postman, codegen, the scalar UI we render below) can consume it.
//
// @Summary      OpenAPI spec (YAML)
// @Description  Returns the embedded OpenAPI 3.x spec. No auth required.
// @Tags         system
// @Produce      application/yaml
// @Success      200  {string}  string
// @Router       /api/openapi.yaml [get]
func openapiYAML(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", openapiSpec)
}

// scalarDocs renders the Scalar API reference UI at /api/docs. It points at
// the sibling /api/openapi.yaml route so any change to the spec is reflected
// without rebuilding the HTML wrapper.
//
// @Summary      API reference (HTML)
// @Description  Renders the Scalar API reference UI. No auth required.
// @Tags         system
// @Produce      html
// @Success      200  {string}  string
// @Router       /api/docs [get]
func scalarDocs(c *gin.Context) {
	// Pass the spec inline (SpecContent) instead of SpecURL: scalar-go treats
	// non-http URLs as filesystem paths, which breaks for our embedded spec.
	html, err := scalar.ApiReferenceHTML(&scalar.Options{
		SpecContent: string(openapiSpec),
		DarkMode:    true,
		CustomOptions: scalar.CustomOptions{
			PageTitle: "TeslaMate API",
		},
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "scalar render failed: %s", err.Error())
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
