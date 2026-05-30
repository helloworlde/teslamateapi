package httpapi

import "github.com/gin-gonic/gin"

// HeaderAPIVersion is the response header that carries the build's apiVersion.
const HeaderAPIVersion = "API-Version"

// APIVersionHeader returns middleware that stamps every response with the
// build's API-Version header.
func APIVersionHeader(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(HeaderAPIVersion, version)
		c.Next()
	}
}
