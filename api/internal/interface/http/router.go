package http

import (
	"github.com/gin-gonic/gin"

	"hesab/api/internal/application/adminauth"
	"hesab/api/internal/application/clientauth"
	"hesab/api/internal/application/health"
	"hesab/api/internal/config"
)

// NewRouter builds the Gin engine with all HTTP routes registered.
func NewRouter(healthSvc *health.Service, authSvc *adminauth.Service, tokens adminauth.TokenIssuer, clientSvc *clientauth.Service, clientTokens clientauth.TokenIssuer, cfg config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(CORS(cfg.CORSOrigins))
	_ = r.SetTrustedProxies(nil) // no proxy trust until a real deployment defines one

	h := &healthHandler{svc: healthSvc}
	r.GET("/health", h.get)
	a := &adminAuthHandler{svc: authSvc, tokens: tokens, cfg: cfg}
	g := r.Group("/admin/auth")
	{
		g.POST("/login", a.login)
		g.POST("/login/2fa", a.login2fa)
		g.POST("/refresh", a.refresh)
		g.POST("/logout", a.logout)
		g.POST("/forgot-password", a.forgot)
		g.POST("/reset-password", a.reset)
	}
	p := r.Group("/admin")
	p.Use(AdminAuth(tokens))
	{
		p.GET("/me", a.me)
		p.POST("/2fa/setup", a.setup)
		p.POST("/2fa/activate", a.activate)
		p.POST("/2fa/disable", a.disable)
	}
	ca := &clientAuthHandler{svc: clientSvc, tokens: clientTokens, cfg: cfg}
	cg := r.Group("/client/auth")
	{
		cg.POST("/login", ca.login)
		cg.POST("/login/2fa", ca.login2fa)
		cg.POST("/refresh", ca.refresh)
		cg.POST("/logout", ca.logout)
		cg.POST("/forgot-password", ca.forgot)
		cg.POST("/reset-password", ca.reset)
	}
	cp := r.Group("/client")
	cp.Use(ClientAuth(clientTokens))
	{
		cp.GET("/me", ca.me)
		cp.POST("/2fa/setup", ca.setup)
		cp.POST("/2fa/activate", ca.activate)
		cp.POST("/2fa/disable", ca.disable)
	}

	return r
}
