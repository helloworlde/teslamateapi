package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	scalar "github.com/watchakorn-18k/scalar-go"

	docs "github.com/tobiasehlert/teslamateapi/src/docs"
)

func serveSwaggerDocJSON(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, docs.SwaggerInfo.ReadDoc())
}

func serveScalarAPIReference(c *gin.Context) {
	html, err := scalar.ApiReferenceHTML(&scalar.Options{
		SpecContent: docs.SwaggerInfo.ReadDoc(),
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
