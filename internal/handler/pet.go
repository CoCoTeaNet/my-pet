package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"my-pet/internal/model"
)

var avatarDir = "./data/avatars"

const maxAvatarSize = 2 << 20 // 2MB

type petReq struct {
	Name      string   `json:"name"`
	Breed     string   `json:"breed"`
	Gender    string   `json:"gender"`
	BirthDate string   `json:"birth_date"`
	Weight    *float64 `json:"weight"`
	Notes     string   `json:"notes"`
}

func newID() string { return model.NewID() }

func (h *Handler) CreatePet(c *gin.Context) {
	var req petReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, "请求格式不正确")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		h.fail(c, http.StatusBadRequest, "宠物名字不能为空")
		return
	}
	if len(req.Name) > 100 {
		h.fail(c, http.StatusBadRequest, "名字过长（最多 100 个字符）")
		return
	}
	if req.Gender != "male" && req.Gender != "female" {
		req.Gender = ""
	}
	pet := model.Pet{
		ID:        newID(),
		Name:      strings.TrimSpace(req.Name),
		Breed:     req.Breed,
		Gender:    req.Gender,
		BirthDate: req.BirthDate,
		Notes:     req.Notes,
	}
	if req.Weight != nil {
		pet.Weight = *req.Weight
	}
	if err := h.DB.Create(&pet).Error; err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	c.JSON(http.StatusOK, pet)
}

func (h *Handler) ListPets(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 50)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	var total int64
	h.DB.Model(&model.Pet{}).Count(&total)
	var pets []model.Pet
	h.DB.Order("created_at ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pets)
	c.JSON(http.StatusOK, gin.H{"items": pets, "total": total})
}

func (h *Handler) GetPet(c *gin.Context) {
	var pet model.Pet
	if err := h.DB.First(&pet, "id = ?", c.Param("id")).Error; err != nil {
		h.fail(c, http.StatusNotFound, "宠物不存在或已被删除")
		return
	}
	var stats model.PetStats
	h.DB.Model(&model.FeedRecord{}).Where("pet_id = ?", pet.ID).Count(&stats.FeedCount)
	h.DB.Model(&model.LogEntry{}).Where("pet_id = ?", pet.ID).Count(&stats.LogCount)
	h.DB.Model(&model.Reminder{}).Where("pet_id = ?", pet.ID).Count(&stats.ReminderCount)
	c.JSON(http.StatusOK, gin.H{
		"id": pet.ID, "name": pet.Name, "breed": pet.Breed, "gender": pet.Gender,
		"birth_date": pet.BirthDate, "weight": pet.Weight, "avatar_url": pet.AvatarURL,
		"notes": pet.Notes, "created_at": pet.CreatedAt,
		"feed_count": stats.FeedCount, "log_count": stats.LogCount, "reminder_count": stats.ReminderCount,
	})
}

func (h *Handler) UpdatePet(c *gin.Context) {
	var pet model.Pet
	if err := h.DB.First(&pet, "id = ?", c.Param("id")).Error; err != nil {
		h.fail(c, http.StatusNotFound, "宠物不存在或已被删除")
		return
	}
	var req petReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, "请求格式不正确")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		h.fail(c, http.StatusBadRequest, "宠物名字不能为空")
		return
	}
	if len(req.Name) > 100 {
		h.fail(c, http.StatusBadRequest, "名字过长（最多 100 个字符）")
		return
	}
	if req.Gender != "male" && req.Gender != "female" {
		req.Gender = ""
	}
	pet.Name = strings.TrimSpace(req.Name)
	pet.Breed = req.Breed
	pet.Gender = req.Gender
	pet.BirthDate = req.BirthDate
	pet.Notes = req.Notes
	if req.Weight != nil {
		pet.Weight = *req.Weight
	}
	if err := h.DB.Save(&pet).Error; err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	c.JSON(http.StatusOK, pet)
}

func (h *Handler) DeletePet(c *gin.Context) {
	id := c.Param("id")
	var pet model.Pet
	if err := h.DB.First(&pet, "id = ?", id).Error; err != nil {
		h.fail(c, http.StatusNotFound, "宠物不存在或已被删除")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		tx.Where("pet_id = ?", id).Delete(&model.FeedRecord{})
		tx.Where("pet_id = ?", id).Delete(&model.Reminder{})
		tx.Where("pet_id = ?", id).Delete(&model.LogEntry{})
		return tx.Delete(&model.Pet{}, "id = ?", id).Error
	})
	if err != nil {
		h.fail(c, http.StatusInternalServerError, "删除失败，请重试")
		return
	}
	if pet.AvatarURL != "" {
		name := filepath.Base(pet.AvatarURL)
		os.Remove(filepath.Join(avatarDir, name))
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) UploadAvatar(c *gin.Context) {
	id := c.Param("id")
	var pet model.Pet
	if err := h.DB.First(&pet, "id = ?", id).Error; err != nil {
		h.fail(c, http.StatusNotFound, "宠物不存在或已被删除")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarSize)
	file, err := c.FormFile("file")
	if err != nil {
		h.fail(c, http.StatusBadRequest, "头像上传失败：文件不能超过 2MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		h.fail(c, http.StatusBadRequest, "仅支持 jpg/png/gif/webp 格式的头像")
		return
	}
	if err := os.MkdirAll(avatarDir, 0o755); err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	name := newID() + ext
	dst := filepath.Join(avatarDir, name)
	src, err := file.Open()
	if err != nil {
		h.fail(c, http.StatusBadRequest, "读取头像文件失败")
		return
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		os.Remove(dst)
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	old := pet.AvatarURL
	pet.AvatarURL = "/api/files/avatars/" + name
	h.DB.Model(&pet).Update("avatar_url", pet.AvatarURL)
	if old != "" {
		os.Remove(filepath.Join(avatarDir, filepath.Base(old)))
	}
	c.JSON(http.StatusOK, gin.H{"avatar_url": pet.AvatarURL})
}

func (h *Handler) ServeAvatar(c *gin.Context) {
	name := filepath.Base(c.Param("name"))
	if name == "." || name == "/" || !strings.Contains(name, ".") {
		c.Status(http.StatusNotFound)
		return
	}
	path := filepath.Join(avatarDir, name)
	if _, err := os.Stat(path); err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		c.Header("Content-Type", "image/png")
	case ".gif":
		c.Header("Content-Type", "image/gif")
	case ".webp":
		c.Header("Content-Type", "image/webp")
	default:
		c.Header("Content-Type", "image/jpeg")
	}
	c.File(path)
}
