package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
)

// systemRoot returns the handler for `/`.
//
// @Summary  服务根路径
// @Description 返回服务运行状态与 API 基础路径，用于快速联通性检查。
// @Tags     system
// @Produce  json
// @Success  200 {object} map[string]string "服务运行中"
// @Router   / [get]
func systemRoot(r *gin.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": r.BasePath()})
	}
}

// systemAPIRoot returns the handler for `/api`.
//
// @Summary  /api 根路径
// @Description 返回服务运行状态与 /api 基础路径。
// @Tags     system
// @Produce  json
// @Success  200 {object} map[string]string "服务运行中"
// @Router   /api [get]
func systemAPIRoot(api *gin.RouterGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": api.BasePath()})
	}
}

// systemV1Root returns the handler for `/api/v1`.
//
// @Summary  /api/v1 根路径
// @Description 返回 V1 接口运行状态与基础路径。
// @Tags     system
// @Produce  json
// @Success  200 {object} map[string]string "V1 接口运行中"
// @Router   /v1 [get]
func systemV1Root(v1 *gin.RouterGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi v1 running..", "path": v1.BasePath()})
	}
}

// systemPing simple ping/pong endpoint.
//
// @Summary  Ping 探活
// @Description 简单的 ping/pong 探活接口，用于网络层连通性检测。
// @Tags     system
// @Produce  json
// @Success  200 {object} map[string]string "pong"
// @Router   /ping [get]
func systemPing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

// healthz is a liveness probe.
//
// @Summary  存活探针
// @Description Kubernetes liveness 探针，进程存活则返回 200。
// @Tags     system
// @Produce  json
// @Success  200 {object} map[string]string "服务存活"
// @Router   /healthz [get]
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": http.StatusText(http.StatusOK)})
}

// readyz is a readiness probe.
//
// @Summary  就绪探针
// @Description Kubernetes readiness 探针，数据库连接建立后返回 200，否则 503。
// @Tags     system
// @Produce  json
// @Success  200 {object} map[string]string "服务就绪"
// @Failure  503 {object} map[string]string "尚未就绪"
// @Router   /readyz [get]
func readyz(c *gin.Context) {
	v := apicommon.IsReady.Load()
	if v == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": http.StatusText(http.StatusServiceUnavailable)})
		return
	}
	if ready, _ := v.(bool); !ready {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": http.StatusText(http.StatusServiceUnavailable)})
		return
	}
	apicommon.HandleSuccessResponse(c, "webserver", gin.H{"status": http.StatusText(http.StatusOK)})
}
