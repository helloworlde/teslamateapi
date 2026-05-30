package main

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
	scalar "github.com/watchakorn-18k/scalar-go"
)

// openapiSpec embeds the hand-written OpenAPI 3.1 spec at build time so the
// binary is self-contained — no extra files to ship alongside it. Keep this
// path relative to the package directory; the file lives next to webserver.go.
//
//go:embed openapi.yaml
var openapiSpec []byte

// openapiYAML serves the raw spec at /api/openapi.yaml so external tooling
// (Postman, codegen, the scalar UI we render below) can consume it.
func openapiYAML(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", openapiSpec)
}

// scalarDocs renders the Scalar API reference UI at /api/docs. It points at
// the sibling /api/openapi.yaml route so any change to the spec is reflected
// without rebuilding the HTML wrapper.
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
