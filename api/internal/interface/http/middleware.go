package http

import (
	"github.com/gin-gonic/gin"
	"hesab/api/internal/application/adminauth"
	"net/http"
	"strings"
)

func CORS(origins []string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[strings.TrimSpace(o)] = true
	}
	return func(c *gin.Context) {
		if o := c.GetHeader("Origin"); allowed[o] {
			c.Header("Access-Control-Allow-Origin", o)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}
func AdminAuth(t adminauth.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		v := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if v == c.GetHeader("Authorization") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		id, e := t.ParseAccess(v)
		if e != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set("adminID", id)
		c.Next()
	}
}
