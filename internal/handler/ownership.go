package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"my-pet/internal/middleware"
	"my-pet/internal/model"
)

func currentUserID(c *gin.Context) string {
	claims, _ := c.Get("claims")
	if cl, ok := claims.(*middleware.Claims); ok {
		return cl.UserID
	}
	return ""
}

// petOfCurrentUser 只返回当前用户名下的宠物；别人的宠物一律按不存在处理，避免泄露存在性
func (h *Handler) petOfCurrentUser(c *gin.Context, petID string) (*model.Pet, bool) {
	var pet model.Pet
	if err := h.DB.First(&pet, "id = ? AND owner_id = ?", petID, currentUserID(c)).Error; err != nil {
		h.fail(c, http.StatusNotFound, "宠物不存在或已被删除")
		return nil, false
	}
	return &pet, true
}

func (h *Handler) ownsPet(c *gin.Context, petID string) bool {
	var count int64
	h.DB.Model(&model.Pet{}).Where("id = ? AND owner_id = ?", petID, currentUserID(c)).Count(&count)
	return count > 0
}
