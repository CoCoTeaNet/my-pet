package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"my-pet/internal/model"
)

var foodTypes = map[string]bool{"dry": true, "wet": true, "treat": true}

func (h *Handler) CreateFeed(c *gin.Context) {
	petID := c.Param("id")
	var pet model.Pet
	if err := h.DB.First(&pet, "id = ?", petID).Error; err != nil {
		h.fail(c, http.StatusNotFound, "宠物不存在或已被删除")
		return
	}
	var req struct {
		FoodType string  `json:"food_type"`
		Amount   float64 `json:"amount"`
		Notes    string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, "请求格式不正确")
		return
	}
	if !foodTypes[req.FoodType] {
		h.fail(c, http.StatusBadRequest, "食物类型只能是干粮、湿粮或零食")
		return
	}
	if req.Amount <= 0 {
		h.fail(c, http.StatusBadRequest, "分量必须大于 0 克")
		return
	}
	if req.Amount > 10000 {
		h.fail(c, http.StatusBadRequest, "分量超出合理范围（最多 10000 克）")
		return
	}
	rec := model.FeedRecord{
		ID:         newID(),
		PetID:      petID,
		FoodType:   req.FoodType,
		Amount:     req.Amount,
		RecordedAt: time.Now(),
		Notes:      req.Notes,
	}
	if err := h.DB.Create(&rec).Error; err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	c.JSON(http.StatusOK, rec)
}

type feedGroup struct {
	Date    string             `json:"date"`
	Records []model.FeedRecord `json:"records"`
}

func (h *Handler) ListFeeds(c *gin.Context) {
	petID := c.Param("id")
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 100)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}
	var total int64
	h.DB.Model(&model.FeedRecord{}).Where("pet_id = ?", petID).Count(&total)
	var records []model.FeedRecord
	h.DB.Where("pet_id = ?", petID).Order("recorded_at DESC, created_at DESC, rowid DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&records)
	groups := []feedGroup{}
	var cur *feedGroup
	for _, r := range records {
		date := r.RecordedAt.In(cst).Format("2006-01-02")
		if cur == nil || cur.Date != date {
			groups = append(groups, feedGroup{Date: date, Records: []model.FeedRecord{}})
			cur = &groups[len(groups)-1]
		}
		cur.Records = append(cur.Records, r)
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "groups": groups})
}

func (h *Handler) DeleteFeed(c *gin.Context) {
	res := h.DB.Delete(&model.FeedRecord{}, "id = ?", c.Param("id"))
	if res.Error != nil || res.RowsAffected == 0 {
		h.fail(c, http.StatusNotFound, "该条喂养信息不存在或已被删除")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ---------- 日志 ----------

var logTypes = map[string]bool{"behavior": true, "health": true, "other": true}

func (h *Handler) CreateLog(c *gin.Context) {
	petID := c.Param("id")
	var pet model.Pet
	if err := h.DB.First(&pet, "id = ?", petID).Error; err != nil {
		h.fail(c, http.StatusNotFound, "宠物不存在或已被删除")
		return
	}
	var req struct {
		Content string `json:"content"`
		LogType string `json:"log_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, "请求格式不正确")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		h.fail(c, http.StatusBadRequest, "日志内容不能为空")
		return
	}
	if len(req.Content) > 5000 {
		h.fail(c, http.StatusBadRequest, "日志内容过长（最多 5000 字）")
		return
	}
	if !logTypes[req.LogType] {
		req.LogType = "other"
	}
	entry := model.LogEntry{
		ID:       newID(),
		PetID:    petID,
		Content:  req.Content,
		LogType:  req.LogType,
		LoggedAt: time.Now(),
	}
	if err := h.DB.Create(&entry).Error; err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (h *Handler) ListLogs(c *gin.Context) {
	petID := c.Param("id")
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 50)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	var total int64
	h.DB.Model(&model.LogEntry{}).Where("pet_id = ?", petID).Count(&total)
	var items []model.LogEntry
	h.DB.Where("pet_id = ?", petID).Order("logged_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) DeleteLog(c *gin.Context) {
	res := h.DB.Delete(&model.LogEntry{}, "id = ?", c.Param("id"))
	if res.Error != nil || res.RowsAffected == 0 {
		h.fail(c, http.StatusNotFound, "该条日志不存在或已被删除")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
