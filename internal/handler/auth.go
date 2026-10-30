package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"my-pet/internal/middleware"
	"my-pet/internal/model"
)

type Handler struct {
	DB *gorm.DB
}

func (h *Handler) fail(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

// ---------- 登录锁定 ----------

type lockState struct {
	fails       int
	lockedUntil time.Time
}

var (
	lockMu   sync.Mutex
	lockouts = map[string]*lockState{}
)

const maxFails = 5
const lockDuration = 15 * time.Minute

func checkLock(username string) (locked bool, remaining time.Duration) {
	lockMu.Lock()
	defer lockMu.Unlock()
	s, ok := lockouts[username]
	if !ok || time.Now().After(s.lockedUntil) {
		return false, 0
	}
	return true, time.Until(s.lockedUntil)
}

func recordFail(username string) int {
	lockMu.Lock()
	defer lockMu.Unlock()
	s := lockouts[username]
	if s == nil {
		s = &lockState{}
		lockouts[username] = s
	}
	s.fails++
	if s.fails >= maxFails {
		s.lockedUntil = time.Now().Add(lockDuration)
		s.fails = 0
	}
	return maxFails - s.fails
}

func clearFails(username string) {
	lockMu.Lock()
	defer lockMu.Unlock()
	delete(lockouts, username)
}

// ---------- 认证 ----------

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		h.fail(c, http.StatusBadRequest, "请输入用户名和密码")
		return
	}
	if locked, remaining := checkLock(req.Username); locked {
		m := int(remaining.Minutes())
		s := int(remaining.Seconds()) % 60
		h.fail(c, http.StatusTooManyRequests, "账号已锁定，请于"+itoa(m)+"分"+itoa(s)+"秒后重试")
		return
	}
	var user model.User
	err := h.DB.Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		left := recordFail(req.Username)
		h.fail(c, http.StatusUnauthorized, "用户名或密码错误，还可尝试 "+itoa(left)+" 次")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		left := recordFail(req.Username)
		h.fail(c, http.StatusUnauthorized, "用户名或密码错误，还可尝试 "+itoa(left)+" 次")
		return
	}
	clearFails(req.Username)
	token, err := middleware.GenerateToken(&user)
	if err != nil {
		h.fail(c, http.StatusInternalServerError, "登录凭证生成失败，请重试")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (h *Handler) Me(c *gin.Context) {
	claims := c.MustGet("claims").(*middleware.Claims)
	var user model.User
	if err := h.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
		h.fail(c, http.StatusUnauthorized, "用户不存在，请重新登录")
		return
	}
	c.JSON(http.StatusOK, user)
}

func validatePassword(p string) string {
	if len(p) < 8 {
		return "密码至少 8 位"
	}
	hasLetter, hasDigit := false, false
	for _, r := range p {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "密码需同时包含字母和数字"
	}
	return ""
}

func (h *Handler) ChangePassword(c *gin.Context) {
	claims := c.MustGet("claims").(*middleware.Claims)
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, "请求格式不正确")
		return
	}
	var user model.User
	if err := h.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
		h.fail(c, http.StatusUnauthorized, "用户不存在，请重新登录")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)) != nil {
		h.fail(c, http.StatusBadRequest, "原密码不正确")
		return
	}
	if msg := validatePassword(req.NewPassword); msg != "" {
		h.fail(c, http.StatusBadRequest, msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	h.DB.Model(&user).Update("password_hash", string(hash))
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ---------- 用户管理 ----------

func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		h.fail(c, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}
	if len(req.Username) > 50 {
		h.fail(c, http.StatusBadRequest, "用户名过长（最多 50 个字符）")
		return
	}
	if req.Role != "admin" && req.Role != "member" {
		req.Role = "member"
	}
	if msg := validatePassword(req.Password); msg != "" {
		h.fail(c, http.StatusBadRequest, msg)
		return
	}
	var count int64
	h.DB.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		h.fail(c, http.StatusBadRequest, "该用户名已被占用")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	user := model.User{Username: req.Username, PasswordHash: string(hash), Role: req.Role}
	if err := h.DB.Create(&user).Error; err != nil {
		h.fail(c, http.StatusBadRequest, "该用户名已被占用")
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) ListUsers(c *gin.Context) {
	var users []model.User
	h.DB.Order("created_at ASC").Find(&users)
	c.JSON(http.StatusOK, gin.H{"items": users})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	claims := c.MustGet("claims").(*middleware.Claims)
	id := c.Param("id")
	if id == claims.UserID {
		h.fail(c, http.StatusBadRequest, "不能删除自己")
		return
	}
	res := h.DB.Delete(&model.User{}, "id = ?", id)
	if res.Error != nil || res.RowsAffected == 0 {
		h.fail(c, http.StatusNotFound, "用户不存在或已被删除")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func itoa(n int) string {
	if n < 0 {
		n = 0
	}
	return intToStr(n)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [12]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
