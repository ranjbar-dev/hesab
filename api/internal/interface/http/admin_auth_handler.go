package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	"hesab/api/internal/application/adminauth"
	"hesab/api/internal/config"
	"hesab/api/internal/domain/admin"
	"io"
	"net/http"
	"strconv"
	"time"
)

type adminAuthHandler struct {
	svc    *adminauth.Service
	tokens adminauth.TokenIssuer
	cfg    config.Config
}
type loginReq struct {
	Phone    string `json:"phone_number" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type twoReq struct {
	PendingToken string `json:"pending_token" binding:"required"`
	Code         string `json:"code" binding:"required"`
}
type forgotReq struct {
	Phone string `json:"phone_number" binding:"required"`
}
type resetReq struct {
	Phone    string `json:"phone_number" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"new_password" binding:"required"`
}
type activateReq struct {
	Secret string `json:"secret" binding:"required"`
	Code   string `json:"code" binding:"required"`
}
type disableReq struct {
	Password string `json:"password" binding:"required"`
}
type updateAdminReq struct {
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name" binding:"required"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number" binding:"required"`
	IsMale      bool   `json:"is_male"`
}

func errJSON(c *gin.Context, status int, code, msg string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": msg}})
}
func adminJSON(a admin.Admin) gin.H {
	return gin.H{"id": a.ID, "first_name": a.FirstName, "last_name": a.LastName, "email": a.Email, "phone_number": a.PhoneNumber, "is_male": a.IsMale, "two_fa_enabled": a.TwoFAEnabled(), "created_at": a.CreatedAt, "avatar_url": avatarURL(a)}
}
func avatarURL(a admin.Admin) any { if a.AvatarType == "" { return nil }; return "/admin/avatars/" + strconv.FormatInt(a.ID, 10) }

// The admin SPA calls the API cross-origin (SPA on :3010, API on :8080), so the
// refresh cookie needs SameSite=None to ride along on fetch. SameSite=None
// requires Secure; browsers treat http://localhost as a secure context, so
// COOKIE_SECURE=true works in dev over plain http too.
func (h *adminAuthHandler) setCookie(c *gin.Context, raw string) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("admin_refresh_token", raw, int(h.cfg.RefreshTokenTTL.Seconds()), "/admin/auth", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}
func (h *adminAuthHandler) clearCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("admin_refresh_token", "", -1, "/admin/auth", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}
func refresh(c *gin.Context) string {
	v, e := c.Request.Cookie("admin_refresh_token")
	if e != nil {
		return ""
	}
	return v.Value
}
func (h *adminAuthHandler) login(c *gin.Context) {
	var r loginReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	if _, err := admin.NormalizePhone(r.Phone); err != nil {
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
	h.setCookie(c, v.RefreshToken)
	c.JSON(200, gin.H{"twofa_required": false, "access_token": v.AccessToken, "expires_in": v.ExpiresIn, "admin": adminJSON(v.Admin)})
}
func (h *adminAuthHandler) login2fa(c *gin.Context) {
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
	a, _ := h.svc.Me(c, id)
	h.setCookie(c, v.RefreshToken)
	c.JSON(200, gin.H{"access_token": v.AccessToken, "expires_in": v.ExpiresIn, "admin": adminJSON(a)})
}
func (h *adminAuthHandler) refresh(c *gin.Context) {
	v, e := h.svc.Refresh(c, refresh(c))
	if e != nil {
		errJSON(c, 401, "refresh_invalid", "نشست نامعتبر است")
		return
	}
	h.setCookie(c, v.RefreshToken)
	c.JSON(200, gin.H{"access_token": v.AccessToken, "expires_in": v.ExpiresIn})
}
func (h *adminAuthHandler) logout(c *gin.Context) {
	_ = h.svc.Logout(c, refresh(c))
	h.clearCookie(c)
	c.Status(204)
}
func (h *adminAuthHandler) forgot(c *gin.Context) {
	var r forgotReq
	if c.ShouldBindJSON(&r) != nil || h.svc.ForgotPassword(c, r.Phone) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"message": "اگر شماره ثبت شده باشد، کد ارسال شد"})
}
func (h *adminAuthHandler) reset(c *gin.Context) {
	var r resetReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	e := h.svc.ResetPassword(c, r.Phone, r.Code, r.Password)
	if errors.Is(e, admin.ErrResetCodeInvalid) {
		errJSON(c, 400, "reset_code_invalid", "کد نامعتبر است")
		return
	}
	if e != nil {
		errJSON(c, 400, "validation_error", "رمز عبور نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"message": "رمز عبور تغییر کرد"})
}
func (h *adminAuthHandler) me(c *gin.Context) {
	a, e := h.svc.Me(c, c.GetInt64("adminID"))
	if e != nil {
		errJSON(c, 401, "unauthorized", "دسترسی نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"admin": adminJSON(a)})
}
func (h *adminAuthHandler) updateProfile(c *gin.Context) {
	var r updateAdminReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	a, err := h.svc.UpdateProfile(c, c.GetInt64("adminID"), r.FirstName, r.LastName, r.Email, r.PhoneNumber, r.IsMale)
	if err != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"admin": adminJSON(a)})
}
func (h *adminAuthHandler) uploadAvatar(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		errJSON(c, 400, "validation_error", "تصویر نامعتبر است")
		return
	}
	file, err := f.Open()
	if err != nil {
		errJSON(c, 400, "validation_error", "تصویر نامعتبر است")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || h.svc.SetAvatar(c, c.GetInt64("adminID"), data, f.Header.Get("Content-Type")) != nil {
		errJSON(c, 400, "validation_error", "تصویر باید PNG، JPEG یا WebP و حداکثر ۱ مگابایت باشد")
		return
	}
	c.JSON(200, gin.H{"avatar_url": "/admin/avatars/" + strconv.FormatInt(c.GetInt64("adminID"), 10) + "?v=" + strconv.FormatInt(time.Now().UnixMilli(), 10)})
}
func (h *adminAuthHandler) deleteAvatar(c *gin.Context) {
	if err := h.svc.ClearAvatar(c, c.GetInt64("adminID")); err != nil {
		errJSON(c, 400, "validation_error", "خطا در حذف تصویر")
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *adminAuthHandler) avatarPublic(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.Status(http.StatusNotFound)
		return
	}
	data, contentType, err := h.svc.GetAvatar(c, id)
	if err != nil || contentType == "" {
		c.Status(http.StatusNotFound)
		return
	}
	c.Data(200, contentType, data)
}
func (h *adminAuthHandler) setup(c *gin.Context) {
	s, u, e := h.svc.Setup2FA(c, c.GetInt64("adminID"))
	if e != nil {
		errJSON(c, 400, "validation_error", "خطا")
		return
	}
	c.JSON(200, gin.H{"secret": s, "otpauth_url": u})
}
func (h *adminAuthHandler) activate(c *gin.Context) {
	var r activateReq
	if c.ShouldBindJSON(&r) != nil || h.svc.Activate2FA(c, c.GetInt64("adminID"), r.Secret, r.Code) != nil {
		errJSON(c, 400, "twofa_invalid", "کد نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"enabled": true})
}
func (h *adminAuthHandler) disable(c *gin.Context) {
	var r disableReq
	if c.ShouldBindJSON(&r) != nil || h.svc.Disable2FA(c, c.GetInt64("adminID"), r.Password) != nil {
		errJSON(c, 401, "invalid_credentials", "رمز عبور نادرست است")
		return
	}
	c.JSON(200, gin.H{"enabled": false})
}
