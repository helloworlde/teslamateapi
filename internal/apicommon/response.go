package apicommon

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleErrorResponse logs the error and returns a 200 with {"error": s2}.
// Behaviour preserved verbatim from the v1 endpoint surface.
func HandleErrorResponse(c *gin.Context, s1 string, s2 string, s3 string) {
	log.Println("[error] " + s1 + " - (" + c.Request.RequestURI + "). " + s2 + "; " + s3)
	c.JSON(http.StatusOK, gin.H{"error": s2})
}

// HandleOtherResponse writes a JSON body at httpCode and logs at info level.
func HandleOtherResponse(c *gin.Context, httpCode int, s string, j interface{}) {
	log.Println("[info] " + s + " - (" + c.Request.RequestURI + ") executed successfully.")
	c.JSON(httpCode, j)
}

// HandleSuccessResponse writes a 200 JSON body and logs at info level.
// In debug mode it also dumps the JSON body to the log.
func HandleSuccessResponse(c *gin.Context, s string, j interface{}) {
	if gin.IsDebugging() {
		log.Println("[debug] " + s + " - (" + c.Request.RequestURI + ") returned data:")
		js, _ := json.Marshal(j)
		log.Printf("[debug] %s\n", js)
	}

	log.Println("[info] " + s + " - (" + c.Request.RequestURI + ") executed successfully.")
	c.JSON(http.StatusOK, j)
}
