package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	app "hesab/api/internal/application/business"
	"hesab/api/internal/domain/business"
	"net/http"
	"strconv"
)

type businessesHandler struct{ svc *app.Service }
type businessReq struct {
	Name string `json:"name"`
}
type memberReq struct {
	PhoneNumber string `json:"phone_number"`
	Role        string `json:"role"`
}
type roleReq struct {
	Role string `json:"role"`
}

func businessJSON(b business.Business) gin.H {
	return gin.H{"id": b.ID, "name": b.Name, "owner_user_id": b.OwnerUserID, "created_at": b.CreatedAt}
}
func memberJSON(m business.Member) gin.H {
	return gin.H{"user_id": m.UserID, "first_name": m.FirstName, "last_name": m.LastName, "phone_number": m.PhoneNumber, "role": m.Role, "created_at": m.CreatedAt}
}
func inviteJSON(i business.Invite) gin.H {
	return gin.H{"id": i.ID, "business_id": i.BusinessID, "business_name": i.BusinessName, "role": i.Role, "invited_by_name": i.InvitedByName, "created_at": i.CreatedAt}
}
func outgoingJSON(i business.Invite) gin.H {
	return gin.H{"id": i.ID, "business_id": i.BusinessID, "user_id": i.UserID, "first_name": i.FirstName, "last_name": i.LastName, "phone_number": i.PhoneNumber, "role": i.Role, "created_at": i.CreatedAt}
}
func businessError(c *gin.Context, e error) {
	switch {
	case errors.Is(e, business.ErrNotFound), errors.Is(e, business.ErrNotMember), errors.Is(e, business.ErrInviteNotFound):
		errJSON(c, 404, "not_found", "موردی یافت نشد")
	case errors.Is(e, business.ErrForbidden):
		errJSON(c, 403, "forbidden", "دسترسی ندارید")
	case errors.Is(e, business.ErrNameRequired):
		errJSON(c, 400, "validation_error", "نام کسب‌وکار الزامی است")
	case errors.Is(e, business.ErrInvalidRole):
		errJSON(c, 400, "invalid_role", "نقش نامعتبر است")
	case errors.Is(e, business.ErrInviteeNotRegistered):
		errJSON(c, 404, "user_not_registered", "کاربر با این شماره در سامانه ثبت‌نام نکرده است")
	case errors.Is(e, business.ErrAlreadyMember):
		errJSON(c, 409, "already_member", "کاربر پیش‌تر عضو این کسب‌وکار است")
	case errors.Is(e, business.ErrInvitePending):
		errJSON(c, 409, "invite_pending", "دعوت در انتظار پاسخ است")
	case errors.Is(e, business.ErrCannotTargetOwner):
		errJSON(c, 409, "owner_immutable", "نقش مالک قابل تغییر نیست")
	case errors.Is(e, business.ErrOwnerCannotLeave):
		errJSON(c, 409, "owner_cannot_leave", "مالک نمی‌تواند از کسب‌وکار خارج شود")
	default:
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
	}
}
func clientID(c *gin.Context) int64 { return c.GetInt64("userID") }
func paramID(c *gin.Context, k string) (int64, bool) {
	id, e := strconv.ParseInt(c.Param(k), 10, 64)
	if e != nil || id < 1 {
		errJSON(c, 400, "validation_error", "ورودی نامعتبر است")
		return 0, false
	}
	return id, true
}
func (h *businessesHandler) list(c *gin.Context) {
	v, e := h.svc.List(c, clientID(c))
	if e != nil {
		businessError(c, e)
		return
	}
	o := make([]gin.H, len(v))
	for i, x := range v {
		o[i] = businessJSON(x.Business)
		o[i]["role"] = x.Role
	}
	c.JSON(200, gin.H{"businesses": o})
}
func (h *businessesHandler) create(c *gin.Context) {
	var r businessReq
	if c.ShouldBindJSON(&r) != nil {
		businessError(c, business.ErrNameRequired)
		return
	}
	v, e := h.svc.Create(c, clientID(c), r.Name)
	if e != nil {
		businessError(c, e)
		return
	}
	c.JSON(201, gin.H{"business": businessJSON(v)})
}
func (h *businessesHandler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	v, r, e := h.svc.Get(c, clientID(c), id)
	if e != nil {
		businessError(c, e)
		return
	}
	c.JSON(200, gin.H{"business": businessJSON(v), "role": r})
}
func (h *businessesHandler) rename(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var r businessReq
	if c.ShouldBindJSON(&r) != nil {
		businessError(c, business.ErrNameRequired)
		return
	}
	v, e := h.svc.Rename(c, clientID(c), id, r.Name)
	if e != nil {
		businessError(c, e)
		return
	}
	c.JSON(200, gin.H{"business": businessJSON(v)})
}
func (h *businessesHandler) delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if e := h.svc.Delete(c, clientID(c), id); e != nil {
		businessError(c, e)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *businessesHandler) members(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	v, r, e := h.svc.Members(c, clientID(c), id)
	if e != nil {
		businessError(c, e)
		return
	}
	o := make([]gin.H, len(v))
	for i, x := range v {
		o[i] = memberJSON(x)
	}
	c.JSON(200, gin.H{"members": o, "role": r})
}
func (h *businessesHandler) invite(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var r memberReq
	if c.ShouldBindJSON(&r) != nil {
		businessError(c, business.ErrInvalidRole)
		return
	}
	if e := h.svc.Invite(c, clientID(c), id, r.PhoneNumber, r.Role); e != nil {
		businessError(c, e)
		return
	}
	c.Status(200)
}
func (h *businessesHandler) removeMember(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, ok := paramID(c, "userId")
	if !ok {
		return
	}
	if e := h.svc.RemoveMember(c, clientID(c), id, u); e != nil {
		businessError(c, e)
		return
	}
	c.Status(204)
}
func (h *businessesHandler) changeRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, ok := paramID(c, "userId")
	if !ok {
		return
	}
	var r roleReq
	if c.ShouldBindJSON(&r) != nil {
		businessError(c, business.ErrInvalidRole)
		return
	}
	if e := h.svc.ChangeRole(c, clientID(c), id, u, r.Role); e != nil {
		businessError(c, e)
		return
	}
	c.Status(204)
}
func (h *businessesHandler) outgoing(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	v, e := h.svc.OutgoingInvites(c, clientID(c), id)
	if e != nil {
		businessError(c, e)
		return
	}
	o := make([]gin.H, len(v))
	for i, x := range v {
		o[i] = outgoingJSON(x)
	}
	c.JSON(200, gin.H{"invites": o})
}
func (h *businessesHandler) cancel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	i, ok := paramID(c, "inviteId")
	if !ok {
		return
	}
	if e := h.svc.CancelInvite(c, clientID(c), id, i); e != nil {
		businessError(c, e)
		return
	}
	c.Status(204)
}
func (h *businessesHandler) incoming(c *gin.Context) {
	v, e := h.svc.PendingInvites(c, clientID(c))
	if e != nil {
		businessError(c, e)
		return
	}
	o := make([]gin.H, len(v))
	for i, x := range v {
		o[i] = inviteJSON(x)
	}
	c.JSON(200, gin.H{"invites": o})
}
func (h *businessesHandler) accept(c *gin.Context) {
	i, ok := parseID(c)
	if !ok {
		return
	}
	if e := h.svc.AcceptInvite(c, clientID(c), i); e != nil {
		businessError(c, e)
		return
	}
	c.Status(204)
}
func (h *businessesHandler) reject(c *gin.Context) {
	i, ok := parseID(c)
	if !ok {
		return
	}
	if e := h.svc.RejectInvite(c, clientID(c), i); e != nil {
		businessError(c, e)
		return
	}
	c.Status(204)
}
