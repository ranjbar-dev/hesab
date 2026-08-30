package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"hesab/api/internal/application/usersadmin"
	"hesab/api/internal/domain/user"
)

type usersAdminHandler struct{ svc *usersadmin.Service }
type createUserReq struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	NationalID  string `json:"national_id"`
	AccountType string `json:"account_type"`
	Password    string `json:"password"`
}
type profileReq struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	NationalID  string `json:"national_id"`
	AccountType string `json:"account_type"`
}
type statusReq struct {
	Status string `json:"status"`
}
type passwordReq struct {
	Password string `json:"new_password"`
}

func userAdminJSON(u user.User) gin.H {
	return gin.H{"id": u.ID, "first_name": u.FirstName, "last_name": u.LastName, "email": u.Email, "phone_number": u.PhoneNumber, "national_id": nullableNationalID(u.NationalID), "account_type": u.AccountType, "status": u.Status, "created_at": u.CreatedAt}
}
func nullableNationalID(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func parseID(c *gin.Context) (int64, bool) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id < 1 {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return 0, false
	}
	return id, true
}
func optional(c *gin.Context, key string) string { return strings.TrimSpace(c.Query(key)) }
func parseTime(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, e := time.Parse(layout, v); e == nil {
			return &t, nil
		}
	}
	return nil, errors.New("invalid time")
}
func (h *usersAdminHandler) list(c *gin.Context) {
	from, e := parseTime(optional(c, "created_from"))
	if e != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	to, e := parseTime(optional(c, "created_to"))
	if e != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	status := optional(c, "status")
	if status != "" && !user.ValidStatus(status) {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	page, size := 1, 20
	if v := optional(c, "page"); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			page = n
		}
	}
	if v := optional(c, "page_size"); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			size = n
		}
	}
	users, total, e := h.svc.List(c, usersadmin.ListFilter{FirstName: optional(c, "first_name"), LastName: optional(c, "last_name"), Phone: optional(c, "phone"), Status: status, CreatedFrom: from, CreatedTo: to}, page, size)
	if e != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	if size > 100 {
		size = 100
	}
	out := make([]gin.H, len(users))
	for i, u := range users {
		out[i] = userAdminJSON(u)
	}
	c.JSON(http.StatusOK, gin.H{"users": out, "total": total, "page": page, "page_size": size})
}
func (h *usersAdminHandler) create(c *gin.Context) {
	var r createUserReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	u, e := h.svc.Create(c, usersadmin.NewUser{FirstName: r.FirstName, LastName: r.LastName, Email: r.Email, PhoneNumber: r.PhoneNumber, NationalID: r.NationalID, AccountType: r.AccountType, Password: r.Password})
	if errors.Is(e, user.ErrPhoneTaken) {
		errJSON(c, 409, "phone_taken", "این شماره موبایل قبلاً ثبت شده است")
		return
	}
	if errors.Is(e, user.ErrWeakPassword) {
		errJSON(c, 400, "weak_password", "رمز عبور باید حداقل ۸ نویسه و شامل حرف و رقم باشد")
		return
	}
	if e != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": userAdminJSON(u)})
}
func (h *usersAdminHandler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, e := h.svc.Get(c, id)
	if errors.Is(e, user.ErrUserNotFound) {
		errJSON(c, 404, "not_found", "کاربر یافت نشد")
		return
	}
	if e != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"user": userAdminJSON(u)})
}
func (h *usersAdminHandler) update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var r profileReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	u, e := h.svc.UpdateProfile(c, id, usersadmin.Profile{FirstName: r.FirstName, LastName: r.LastName, Email: r.Email, NationalID: r.NationalID, AccountType: r.AccountType})
	if errors.Is(e, user.ErrUserNotFound) {
		errJSON(c, 404, "not_found", "کاربر یافت نشد")
		return
	}
	if e != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"user": userAdminJSON(u)})
}
func (h *usersAdminHandler) setStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var r statusReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	u, e := h.svc.SetStatus(c, id, r.Status)
	if errors.Is(e, user.ErrUserNotFound) {
		errJSON(c, 404, "not_found", "کاربر یافت نشد")
		return
	}
	if e != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	c.JSON(200, gin.H{"user": userAdminJSON(u)})
}
func (h *usersAdminHandler) resetPassword(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var r passwordReq
	if c.ShouldBindJSON(&r) != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	e := h.svc.ResetPassword(c, id, r.Password)
	if errors.Is(e, user.ErrUserNotFound) {
		errJSON(c, 404, "not_found", "کاربر یافت نشد")
		return
	}
	if errors.Is(e, user.ErrWeakPassword) {
		errJSON(c, 400, "weak_password", "رمز عبور باید حداقل ۸ نویسه و شامل حرف و رقم باشد")
		return
	}
	if e != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *usersAdminHandler) remove(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	e := h.svc.Delete(c, id)
	if errors.Is(e, user.ErrUserNotFound) {
		errJSON(c, 404, "not_found", "کاربر یافت نشد")
		return
	}
	if e != nil {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return
	}
	c.Status(http.StatusNoContent)
}
