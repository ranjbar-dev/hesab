package http

import (
	"github.com/gin-gonic/gin"

	"hesab/api/internal/application/adminauth"
	"hesab/api/internal/application/business"
	"hesab/api/internal/application/businessadmin"
	"hesab/api/internal/application/clientauth"
	"hesab/api/internal/application/health"
	"hesab/api/internal/application/usersadmin"
	"hesab/api/internal/config"
)

// NewRouter builds the Gin engine with all HTTP routes registered.
func NewRouter(healthSvc *health.Service, authSvc *adminauth.Service, tokens adminauth.TokenIssuer, usersAdminSvc *usersadmin.Service, businessAdminSvc *businessadmin.Service, businessSvc *business.Service, clientSvc *clientauth.Service, clientTokens clientauth.TokenIssuer, cfg config.Config) *gin.Engine {
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
		ua := &usersAdminHandler{svc: usersAdminSvc}
		ba := &businessesAdminHandler{svc: businessAdminSvc}
		p.GET("/users", ua.list)
		p.POST("/users", ua.create)
		p.GET("/users/:id", ua.get)
		p.PATCH("/users/:id", ua.update)
		p.POST("/users/:id/status", ua.setStatus)
		p.POST("/users/:id/reset-password", ua.resetPassword)
		p.DELETE("/users/:id", ua.remove)
		p.GET("/users/:id/businesses", ba.userBusinesses)
		p.GET("/businesses", ba.list)
		p.POST("/businesses", ba.create)
		p.GET("/businesses/:id", ba.get)
		p.PATCH("/businesses/:id", ba.rename)
		p.DELETE("/businesses/:id", ba.delete)
		p.POST("/businesses/:id/members", ba.addMember)
		p.PATCH("/businesses/:id/members/:userId", ba.changeRole)
		p.DELETE("/businesses/:id/members/:userId", ba.removeMember)
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
		b := &businessesHandler{svc: businessSvc}
		cp.GET("/businesses", b.list)
		cp.POST("/businesses", b.create)
		cp.GET("/businesses/:id", b.get)
		cp.PATCH("/businesses/:id", b.rename)
		cp.DELETE("/businesses/:id", b.delete)
		cp.GET("/businesses/:id/members", b.members)
		cp.POST("/businesses/:id/members", b.invite)
		cp.DELETE("/businesses/:id/members/:userId", b.removeMember)
		cp.PATCH("/businesses/:id/members/:userId", b.changeRole)
		cp.GET("/businesses/:id/invites", b.outgoing)
		cp.DELETE("/businesses/:id/invites/:inviteId", b.cancel)
		cp.GET("/invites", b.incoming)
		cp.POST("/invites/:id/accept", b.accept)
		cp.POST("/invites/:id/reject", b.reject)
		cp.GET("/me", ca.me)
		cp.POST("/2fa/setup", ca.setup)
		cp.POST("/2fa/activate", ca.activate)
		cp.POST("/2fa/disable", ca.disable)
	}

	return r
}
