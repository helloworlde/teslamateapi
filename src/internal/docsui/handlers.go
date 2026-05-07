package docsui

import (
	"net/http"

	"github.com/gin-gonic/gin"
	scalar "github.com/watchakorn-18k/scalar-go"

	docs "github.com/tobiasehlert/teslamateapi/src/docs"
)

// RegisterRoutes 注册 OpenAPI JSON、Scalar UI 和旧 Swagger 入口。
func RegisterRoutes(group *gin.RouterGroup, basePath string) {
	group.GET("/docs", func(c *gin.Context) { serveScalarAPIReference(c, basePath) })
	group.GET("/docs/openapi.json", serveOpenAPIDocumentJSON)
	group.GET("/docs/swagger", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePath+"/docs/swagger/index.html") })
	group.GET("/docs/swagger/index.html", func(c *gin.Context) { serveScalarAPIReference(c, basePath) })
	group.GET("/docs/swagger/doc.json", serveSwaggerDocJSON)
}

func serveOpenAPIDocumentJSON(c *gin.Context) {
	serveSwaggerDocJSON(c)
}

func serveSwaggerDocJSON(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, docs.SwaggerInfo.ReadDoc())
}

func serveScalarAPIReference(c *gin.Context, basePath string) {
	html, err := scalar.ApiReferenceHTML(&scalar.Options{
		SpecURL:       basePath + "/docs/openapi.json",
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
