package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"my-pet/internal/model"
)

var alertsFile = "./data/alerts.log"

func getSetting(h *Handler) model.Setting {
	var s model.Setting
	h.DB.First(&s, "id = ?", 1)
	if s.ReminderIntervalMinutes < 1 {
		s.ReminderIntervalMinutes = 1
	}
	if s.BackupPath == "" {
		s.BackupPath = "./data/backups"
	}
	return s
}

func (h *Handler) GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, getSetting(h))
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	var req model.Setting
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, "请求格式不正确")
		return
	}
	if req.ReminderIntervalMinutes < 1 {
		h.fail(c, http.StatusBadRequest, "提醒检查间隔至少为 1 分钟")
		return
	}
	s := getSetting(h)
	s.WebhookURL = strings.TrimSpace(req.WebhookURL)
	s.ReminderIntervalMinutes = req.ReminderIntervalMinutes
	s.BackupPath = strings.TrimSpace(req.BackupPath)
	if s.BackupPath == "" {
		s.BackupPath = "./data/backups"
	}
	if err := h.DB.Save(&s).Error; err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) GetWebhook(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"webhook_url": getSetting(h).WebhookURL})
}

func (h *Handler) UpdateWebhook(c *gin.Context) {
	var req struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, "请求格式不正确")
		return
	}
	s := getSetting(h)
	s.WebhookURL = strings.TrimSpace(req.WebhookURL)
	if err := h.DB.Save(&s).Error; err != nil {
		h.fail(c, http.StatusInternalServerError, "保存失败，请重试")
		return
	}
	c.JSON(http.StatusOK, gin.H{"webhook_url": s.WebhookURL})
}

func sendWebhook(url string, payload map[string]interface{}) error {
	if url == "" {
		return fmt.Errorf("未配置 Webhook 地址")
	}
	body, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("对方返回状态码 %d", resp.StatusCode)
	}
	return nil
}

func AppendAlert(message string, success bool) {
	os.MkdirAll(filepath.Dir(alertsFile), 0o755)
	entry, _ := json.Marshal(map[string]interface{}{
		"time":    time.Now().Format(time.RFC3339),
		"message": message,
		"success": success,
	})
	f, err := os.OpenFile(alertsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(entry, '\n'))
}

func (h *Handler) TestWebhook(c *gin.Context) {
	url := getSetting(h).WebhookURL
	if url == "" {
		h.fail(c, http.StatusBadRequest, "请先填写 Webhook 地址")
		return
	}
	if err := sendWebhook(url, map[string]interface{}{
		"message": "这是一条来自《我的宠物》的测试通知",
		"test":    true,
	}); err != nil {
		AppendAlert("Webhook 测试失败："+err.Error(), false)
		h.fail(c, http.StatusBadRequest, "Webhook 发送失败："+err.Error())
		return
	}
	AppendAlert("Webhook 测试发送成功", true)
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) ListNotifications(c *gin.Context) {
	items := []map[string]interface{}{}
	data, err := os.ReadFile(alertsFile)
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		start := 0
		if len(lines) > 20 {
			start = len(lines) - 20
		}
		for i := len(lines) - 1; i >= start; i-- {
			if strings.TrimSpace(lines[i]) == "" {
				continue
			}
			var m map[string]interface{}
			if json.Unmarshal([]byte(lines[i]), &m) == nil {
				items = append(items, m)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Backup(c *gin.Context) {
	s := getSetting(h)
	if err := os.MkdirAll(s.BackupPath, 0o755); err != nil {
		h.fail(c, http.StatusBadRequest, "备份目录创建失败："+err.Error())
		return
	}
	h.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	src := dbFilePath
	if _, err := os.Stat(src); err != nil {
		h.fail(c, http.StatusInternalServerError, "数据库文件不存在")
		return
	}
	name := "my-pet-backup-" + time.Now().Format("20060102-150405") + ".db"
	dst := filepath.Join(s.BackupPath, name)
	in, err := os.Open(src)
	if err != nil {
		h.fail(c, http.StatusInternalServerError, "读取数据库失败："+err.Error())
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		h.fail(c, http.StatusInternalServerError, "写入备份文件失败："+err.Error())
		return
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		h.fail(c, http.StatusInternalServerError, "写入备份文件失败："+err.Error())
		return
	}
	out.Close()
	now := time.Now()
	h.DB.Model(&model.Setting{}).Where("id = ?", 1).Update("last_backup_at", &now)
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Header("Content-Type", "application/octet-stream")
	c.File(dst)
}
