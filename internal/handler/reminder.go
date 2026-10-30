package handler

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"my-pet/internal/model"
)

type reminderReq struct {
	Type         string `json:"type"`
	IntervalDays int    `json:"interval_days"`
	NextDueAt    string `json:"next_due_at"`
	Notes        string `json:"notes"`
	Enabled      *bool  `json:"enabled"`
}

func (h *Handler) CreateReminder(c *gin.Context) {
	petID := c.Param("id")
	var pet model.Pet
	if err := h.DB.First(&pet, "id = ?", petID).Error; err != nil {
		h.fail(c, http.StatusNotFound, "宠物不存在或已被删除")
		return
	}
	var req reminderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, "请求格式不正确")
		return
	}
	due, ok := parseFlexibleTime(req.NextDueAt)
	if !ok {
		h.fail(c, http.StatusBadRequest, "下次执行日期不合法")
		return
	}
	if req.Type == "" {
		h.fail(c, http.StatusBadRequest, "请选择提醒类型")
		return
	}
	if req.IntervalDays <= 0 {
		h.fail(c, http.StatusBadRequest, "重复周期必须大于 0 天")
		return
	}
	r := model.Reminder{
		ID:           newID(),
		PetID:        petID,
		Type:         req.Type,
		IntervalDays: req.IntervalDays,
		NextDueAt:    due,
		Notes:        req.Notes,
		Enabled:      true,
	}
	if err := h.DB.Create(&r).Error; err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	c.JSON(http.StatusOK, withOverdue(r))
}

func withOverdue(r model.Reminder) gin.H {
	return gin.H{
		"id": r.ID, "pet_id": r.PetID, "type": r.Type,
		"interval_days": r.IntervalDays, "last_triggered_at": r.LastTriggeredAt,
		"next_due_at": r.NextDueAt, "notes": r.Notes, "enabled": r.Enabled,
		"created_at": r.CreatedAt,
		"overdue":    r.Enabled && r.NextDueAt.Before(time.Now()),
	}
}

func (h *Handler) ListReminders(c *gin.Context) {
	petID := c.Param("id")
	var items []model.Reminder
	h.DB.Where("pet_id = ?", petID).Find(&items)
	out := make([]gin.H, 0, len(items))
	for _, r := range items {
		out = append(out, withOverdue(r))
	}
	// 过期置顶，其余按 NextDueAt 升序
	sort.SliceStable(out, func(i, j int) bool {
		oi, oj := out[i]["overdue"].(bool), out[j]["overdue"].(bool)
		if oi != oj {
			return oi
		}
		ti, _ := out[i]["next_due_at"].(time.Time)
		tj, _ := out[j]["next_due_at"].(time.Time)
		return ti.Before(tj)
	})
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *Handler) UpdateReminder(c *gin.Context) {
	var r model.Reminder
	if err := h.DB.First(&r, "id = ?", c.Param("id")).Error; err != nil {
		h.fail(c, http.StatusNotFound, "提醒不存在或已被删除")
		return
	}
	var req reminderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, "请求格式不正确")
		return
	}
	if req.Type == "" {
		h.fail(c, http.StatusBadRequest, "请选择提醒类型")
		return
	}
	if req.IntervalDays <= 0 {
		h.fail(c, http.StatusBadRequest, "重复周期必须大于 0 天")
		return
	}
	due, ok := parseFlexibleTime(req.NextDueAt)
	if !ok {
		h.fail(c, http.StatusBadRequest, "下次执行日期不合法")
		return
	}
	r.Type = req.Type
	r.IntervalDays = req.IntervalDays
	r.NextDueAt = due
	r.Notes = req.Notes
	if req.Enabled != nil {
		r.Enabled = *req.Enabled
	}
	if err := h.DB.Save(&r).Error; err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	c.JSON(http.StatusOK, withOverdue(r))
}

func (h *Handler) CompleteReminder(c *gin.Context) {
	var r model.Reminder
	if err := h.DB.First(&r, "id = ?", c.Param("id")).Error; err != nil {
		h.fail(c, http.StatusNotFound, "提醒不存在或已被删除")
		return
	}
	now := time.Now()
	r.LastTriggeredAt = &now
	r.NextDueAt = now.Add(time.Duration(r.IntervalDays) * 24 * time.Hour)
	if err := h.DB.Save(&r).Error; err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	c.JSON(http.StatusOK, withOverdue(r))
}

func (h *Handler) DeleteReminder(c *gin.Context) {
	res := h.DB.Delete(&model.Reminder{}, "id = ?", c.Param("id"))
	if res.Error != nil || res.RowsAffected == 0 {
		h.fail(c, http.StatusNotFound, "提醒不存在或已被删除")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
