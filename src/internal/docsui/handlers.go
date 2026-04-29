package docsui

import (
	"net/http"

	"github.com/gin-gonic/gin"
	scalar "github.com/watchakorn-18k/scalar-go"

	docs "github.com/tobiasehlert/teslamateapi/src/docs"
)

// RegisterRoutes 注册 OpenAPI JSON、Scalar UI 和旧 Swagger 入口。
func RegisterRoutes(v1 *gin.RouterGroup, basePathV1 string) {
	// 文档路由统一挂在 /api/v1/docs 下，Swagger 旧入口跳转到 Scalar UI。
	v1.GET("/docs", serveScalarAPIReference)
	v1.GET("/docs/openapi.json", serveOpenAPIDocumentJSON)
	v1.GET("/docs/swagger", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePathV1+"/docs/swagger/index.html") })
	v1.GET("/docs/swagger/index.html", serveScalarAPIReference)
	v1.GET("/docs/swagger/doc.json", serveSwaggerDocJSON)
}

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
