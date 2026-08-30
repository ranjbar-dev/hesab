package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hesab/api/internal/application/health"
)

type healthHandler struct {
	svc *health.Service
}

// get handles GET /health. Returns 200 when every dependency is reachable,
// 503 otherwise.
func (h *healthHandler) get(c *gin.Context) {
	st := h.svc.Check(c.Request.Context())

	code := http.StatusOK
	database := "up"
	status := "ok"
	if !st.OK() {
		code = http.StatusServiceUnavailable
		database = "down"
		status = "degraded"
	}

	c.JSON(code, gin.H{"status": status, "database": database})
}
