package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	scalar "github.com/watchakorn-18k/scalar-go"

	"github.com/tobiasehlert/teslamateapi/docs"
)

// OpenAPIYAML serves the embedded OpenAPI spec at /api/openapi.yaml so
// external tooling (Postman, codegen, the scalar UI we render below) can
// consume it.
//
// @Summary      OpenAPI spec (YAML)
// @Description  Returns the embedded OpenAPI 3.x spec. No auth required.
// @Tags         system
// @Produce      application/yaml
// @Success      200  {string}  string
// @Router       /api/openapi.yaml [get]
func OpenAPIYAML(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.Spec)
}

// ScalarDocs renders the Scalar API reference UI at /api/docs. It points
// at the sibling /api/openapi.yaml route so any change to the spec is
// reflected without rebuilding the HTML wrapper.
//
// @Summary      API reference (HTML)
// @Description  Renders the Scalar API reference UI. No auth required.
// @Tags         system
// @Produce      html
// @Success      200  {string}  string
// @Router       /api/docs [get]
func ScalarDocs(c *gin.Context) {
	html, err := scalar.ApiReferenceHTML(&scalar.Options{
		SpecContent: string(docs.Spec),
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
