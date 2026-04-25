package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	scalar "github.com/watchakorn-18k/scalar-go"

	docs "github.com/tobiasehlert/teslamateapi/src/docs"
)

func serveOpenAPIDocumentJSON(c *gin.Context) {
	serveSwaggerDocJSON(c)
}

func serveSwaggerDocJSON(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, docs.SwaggerInfo.ReadDoc())
}

func serveScalarAPIReference(c *gin.Context) {
	html, err := scalar.ApiReferenceHTML(&scalar.Options{
		SpecURL:       "/api/v1/docs/openapi.json",
		SpecContent:   docs.SwaggerInfo.ReadDoc(),
		Theme:         scalar.ThemeDefault,
		Layout:        scalar.LayoutModern,
		BaseServerURL: "/api",
		CustomOptions: scalar.CustomOptions{
			PageTitle: "TeslaMateApi",
		},
	})
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
