package cronjob

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"my-pet/internal/handler"
	"my-pet/internal/model"
)

// notifiedDue 记录每个提醒已通知过的 NextDueAt，避免同一到期时间反复推送
var notifiedDue = map[string]time.Time{}

func Start(db *gorm.DB) {
	go func() {
		for {
			interval := intervalMinutes(db)
			runOnce(db)
			time.Sleep(time.Duration(interval) * time.Minute)
		}
	}()
}

func intervalMinutes(db *gorm.DB) int {
	var s model.Setting
	db.First(&s, "id = ?", 1)
	if s.ReminderIntervalMinutes < 1 {
		return 1
	}
	return s.ReminderIntervalMinutes
}

func runOnce(db *gorm.DB) {
	now := time.Now()
	var reminders []model.Reminder
	if err := db.Where("enabled = ? AND next_due_at <= ?", true, now).Find(&reminders).Error; err != nil {
		return
	}
	for _, r := range reminders {
		if last, ok := notifiedDue[r.ID]; ok && !r.NextDueAt.After(last) {
			continue
		}
		notifiedDue[r.ID] = r.NextDueAt
		var pet model.Pet
		petName := "宠物"
		if err := db.First(&pet, "id = ?", r.PetID).Error; err == nil {
			petName = pet.Name
		}
		dueStr := r.NextDueAt.In(time.FixedZone("CST", 8*3600)).Format("01月02日 15:04")
		msg := fmt.Sprintf("健康提醒到期：%s 的「%s」应于 %s 执行", petName, r.Type, dueStr)
		if r.Notes != "" {
			msg += "（" + r.Notes + "）"
		}
		log.Printf("[提醒] %s", msg)
		var s model.Setting
		db.First(&s, "id = ?", 1)
		if s.WebhookURL != "" {
			if err := handler.PostWebhook(s.WebhookURL, map[string]interface{}{
				"message":     msg,
				"pet_name":    petName,
				"pet_id":      r.PetID,
				"reminder_id": r.ID,
				"type":        r.Type,
				"next_due_at": r.NextDueAt,
			}); err != nil {
				handler.AppendAlert(msg+" | Webhook 发送失败："+err.Error(), false)
			} else {
				handler.AppendAlert(msg, true)
			}
		} else {
			handler.AppendAlert(msg, true)
		}
	}
}
