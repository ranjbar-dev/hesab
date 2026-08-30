// ponytail: parallel to adminauth, kept separate on purpose.
package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	"hesab/api/internal/application/clientauth"
	"hesab/api/internal/config"
	"hesab/api/internal/domain/user"
	"net/http"
)

type clientAuthHandler struct {
	svc    *clientauth.Service
	tokens clientauth.TokenIssuer
	cfg    config.Config
}

func userJSON(u user.User) gin.H {
	return gin.H{"id": u.ID, "first_name": u.FirstName, "last_name": u.LastName, "email": u.Email, "phone_number": u.PhoneNumber, "two_fa_enabled": u.TwoFAEnabled(), "created_at": u.CreatedAt}
}
func (h *clientAuthHandler) setClientCookie(c *gin.Context, raw string) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("client_refresh_token", raw, int(h.cfg.RefreshTokenTTL.Seconds()), "/client/auth", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}
func (h *clientAuthHandler) clearClientCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("client_refresh_token", "", -1, "/client/auth", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}
func clientRefresh(c *gin.Context) string {
	v, e := c.Request.Cookie("client_refresh_token")
	if e != nil {
		return ""
	}
	return v.Value
}
func (h *clientAuthHandler) login(c *gin.Context) {
	var r loginReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	if _, err := user.NormalizePhone(r.Phone); err != nil {
		errJSON(c, 400, "validation_error", "شماره موبایل نامعتبر است")
		return
	}
	v, e := h.svc.Login(c, r.Phone, r.Password)
	if e != nil {
		errJSON(c, 401, "invalid_credentials", "اطلاعات ورود نادرست است")
		return
	}
	if v.TwoFARequired {
		c.JSON(200, gin.H{"twofa_required": true, "pending_token": v.PendingToken})
		return
	}
	h.setClientCookie(c, v.RefreshToken)
	c.JSON(200, gin.H{"twofa_required": false, "access_token": v.AccessToken, "expires_in": v.ExpiresIn, "user": userJSON(v.User)})
}
func (h *clientAuthHandler) login2fa(c *gin.Context) {
	var r twoReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	v, e := h.svc.LoginVerify2FA(c, r.PendingToken, r.Code)
	if e != nil {
		errJSON(c, 401, "twofa_invalid", "کد نامعتبر است")
		return
	}
	id, _ := h.tokens.ParsePending(r.PendingToken)
	u, _ := h.svc.Me(c, id)
	h.setClientCookie(c, v.RefreshToken)
	c.JSON(200, gin.H{"access_token": v.AccessToken, "expires_in": v.ExpiresIn, "user": userJSON(u)})
}
func (h *clientAuthHandler) refresh(c *gin.Context) {
	v, e := h.svc.Refresh(c, clientRefresh(c))
	if e != nil {
		errJSON(c, 401, "refresh_invalid", "نشست نامعتبر است")
		return
	}
	h.setClientCookie(c, v.RefreshToken)
	c.JSON(200, gin.H{"access_token": v.AccessToken, "expires_in": v.ExpiresIn})
}
func (h *clientAuthHandler) logout(c *gin.Context) {
	_ = h.svc.Logout(c, clientRefresh(c))
	h.clearClientCookie(c)
	c.Status(204)
}
func (h *clientAuthHandler) forgot(c *gin.Context) {
	var r forgotReq
	if c.ShouldBindJSON(&r) != nil || h.svc.ForgotPassword(c, r.Phone) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"message": "اگر شماره ثبت شده باشد، کد ارسال شد"})
}
func (h *clientAuthHandler) reset(c *gin.Context) {
	var r resetReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	e := h.svc.ResetPassword(c, r.Phone, r.Code, r.Password)
	if errors.Is(e, user.ErrResetCodeInvalid) {
		errJSON(c, 400, "reset_code_invalid", "کد نامعتبر است")
		return
	}
	if e != nil {
		errJSON(c, 400, "validation_error", "رمز عبور نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"message": "رمز عبور تغییر کرد"})
}
func (h *clientAuthHandler) me(c *gin.Context) {
	u, e := h.svc.Me(c, c.GetInt64("userID"))
	if e != nil {
		errJSON(c, 401, "unauthorized", "دسترسی نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"user": userJSON(u)})
}
func (h *clientAuthHandler) setup(c *gin.Context) {
	s, u, e := h.svc.Setup2FA(c, c.GetInt64("userID"))
	if e != nil {
		errJSON(c, 400, "validation_error", "خطا")
		return
	}
	c.JSON(200, gin.H{"secret": s, "otpauth_url": u})
}
func (h *clientAuthHandler) activate(c *gin.Context) {
	var r activateReq
	if c.ShouldBindJSON(&r) != nil || h.svc.Activate2FA(c, c.GetInt64("userID"), r.Secret, r.Code) != nil {
		errJSON(c, 400, "twofa_invalid", "کد نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"enabled": true})
}
func (h *clientAuthHandler) disable(c *gin.Context) {
	var r disableReq
	if c.ShouldBindJSON(&r) != nil || h.svc.Disable2FA(c, c.GetInt64("userID"), r.Password) != nil {
		errJSON(c, 401, "invalid_credentials", "رمز عبور نادرست است")
		return
	}
	c.JSON(200, gin.H{"enabled": false})
}
