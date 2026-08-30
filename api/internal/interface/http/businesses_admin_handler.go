package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	app "hesab/api/internal/application/businessadmin"
	"hesab/api/internal/domain/business"
	"hesab/api/internal/domain/user"
	"net/http"
	"strconv"
)

type businessesAdminHandler struct{ svc *app.Service }
type adminCreateBusinessReq struct {
	Name        string `json:"name"`
	OwnerUserID int64  `json:"owner_user_id"`
}

func ownerJSON(o app.Owner) gin.H {
	return gin.H{"id": o.ID, "first_name": o.FirstName, "last_name": o.LastName, "phone_number": o.PhoneNumber}
}
func adminBusinessError(c *gin.Context, e error) {
	if errors.Is(e, user.ErrUserNotFound) {
		errJSON(c, 404, "not_found", "کاربر یافت نشد")
		return
	}
	businessError(c, e)
}
func (h *businessesAdminHandler) list(c *gin.Context) {
	p, s := 1, 20
	if n, e := strconv.Atoi(c.Query("page")); e == nil && n > 0 {
		p = n
	}
	if n, e := strconv.Atoi(c.Query("page_size")); e == nil && n > 0 {
		s = n
	}
	v, t, e := h.svc.List(c, optional(c, "name"), p, s)
	if e != nil {
		adminBusinessError(c, e)
		return
	}
	if s > 100 {
		s = 100
	}
	o := make([]gin.H, len(v))
	for i, x := range v {
		o[i] = businessJSON(x.Business)
		o[i]["member_count"] = x.MemberCount
		o[i]["owner"] = ownerJSON(x.Owner)
	}
	c.JSON(200, gin.H{"businesses": o, "total": t, "page": p, "page_size": s})
}
func (h *businessesAdminHandler) create(c *gin.Context) {
	var r adminCreateBusinessReq
	if c.ShouldBindJSON(&r) != nil {
		adminBusinessError(c, business.ErrNameRequired)
		return
	}
	v, e := h.svc.Create(c, r.OwnerUserID, r.Name)
	if e != nil {
		adminBusinessError(c, e)
		return
	}
	c.JSON(201, gin.H{"business": businessJSON(v)})
}
func (h *businessesAdminHandler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	b, o, m, e := h.svc.Get(c, id)
	if e != nil {
		adminBusinessError(c, e)
		return
	}
	ms := make([]gin.H, len(m))
	for i, x := range m {
		ms[i] = memberJSON(x)
	}
	c.JSON(200, gin.H{"business": businessJSON(b), "owner": ownerJSON(o), "members": ms})
}
func (h *businessesAdminHandler) rename(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var r businessReq
	if c.ShouldBindJSON(&r) != nil {
		adminBusinessError(c, business.ErrNameRequired)
		return
	}
	v, e := h.svc.Rename(c, id, r.Name)
	if e != nil {
		adminBusinessError(c, e)
		return
	}
	c.JSON(200, gin.H{"business": businessJSON(v)})
}
func (h *businessesAdminHandler) delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if e := h.svc.Delete(c, id); e != nil {
		adminBusinessError(c, e)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *businessesAdminHandler) addMember(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var r memberReq
	if c.ShouldBindJSON(&r) != nil {
		adminBusinessError(c, business.ErrInvalidRole)
		return
	}
	m, e := h.svc.AddMemberByPhone(c, id, r.PhoneNumber, r.Role)
	if e != nil {
		adminBusinessError(c, e)
		return
	}
	c.JSON(201, gin.H{"member": memberJSON(m)})
}
func (h *businessesAdminHandler) changeRole(c *gin.Context) {
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
		adminBusinessError(c, business.ErrInvalidRole)
		return
	}
	if e := h.svc.ChangeRole(c, id, u, r.Role); e != nil {
		adminBusinessError(c, e)
		return
	}
	c.Status(204)
}
func (h *businessesAdminHandler) removeMember(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, ok := paramID(c, "userId")
	if !ok {
		return
	}
	if e := h.svc.RemoveMember(c, id, u); e != nil {
		adminBusinessError(c, e)
		return
	}
	c.Status(204)
}
func (h *businessesAdminHandler) userBusinesses(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	o, j, e := h.svc.UserBusinesses(c, id)
	if e != nil {
		adminBusinessError(c, e)
		return
	}
	owned := make([]gin.H, len(o))
	for i, x := range o {
		owned[i] = gin.H{"id": x.ID, "name": x.Name, "member_count": x.MemberCount, "created_at": x.CreatedAt}
	}
	joined := make([]gin.H, len(j))
	for i, x := range j {
		joined[i] = gin.H{"id": x.ID, "name": x.Name, "role": x.Role, "owner_name": x.OwnerName, "created_at": x.CreatedAt}
	}
	c.JSON(200, gin.H{"owned": owned, "joined": joined})
}
