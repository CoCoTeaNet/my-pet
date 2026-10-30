package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

func NewID() string {
	b := make([]byte, 16)
	rand.Read(b)
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = NewID()
	}
	return nil
}

type Pet struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Breed     string    `gorm:"size:100" json:"breed"`
	Gender    string    `gorm:"size:10" json:"gender"`
	BirthDate string    `gorm:"size:20" json:"birth_date"`
	Weight    float64   `json:"weight"`
	AvatarURL string    `gorm:"size:255" json:"avatar_url"`
	Notes     string    `gorm:"type:text" json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PetStats struct {
	FeedCount     int64 `json:"feed_count"`
	LogCount      int64 `json:"log_count"`
	ReminderCount int64 `json:"reminder_count"`
}

type FeedRecord struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	PetID      string    `gorm:"index;size:36;not null" json:"pet_id"`
	FoodType   string    `gorm:"size:10;not null" json:"food_type"`
	Amount     float64   `json:"amount"`
	RecordedAt time.Time `json:"recorded_at"`
	Notes      string    `gorm:"type:text" json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

type Reminder struct {
	ID              string     `gorm:"primaryKey;size:36" json:"id"`
	PetID           string     `gorm:"index;size:36;not null" json:"pet_id"`
	Type            string     `gorm:"size:20;not null" json:"type"`
	IntervalDays    int        `json:"interval_days"`
	LastTriggeredAt *time.Time `json:"last_triggered_at"`
	NextDueAt       time.Time  `json:"next_due_at"`
	Notes           string     `gorm:"type:text" json:"notes"`
	Enabled         bool       `gorm:"default:true" json:"enabled"`
	CreatedAt       time.Time  `json:"created_at"`
}

type LogEntry struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	PetID     string    `gorm:"index;size:36;not null" json:"pet_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	LogType   string    `gorm:"size:20" json:"log_type"`
	LoggedAt  time.Time `json:"logged_at"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	PasswordHash string    `gorm:"size:100;not null" json:"-"`
	Role         string    `gorm:"size:10;not null" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Setting struct {
	ID                      uint       `gorm:"primaryKey" json:"-"`
	WebhookURL              string     `gorm:"size:500" json:"webhook_url"`
	ReminderIntervalMinutes int        `json:"reminder_interval_minutes"`
	BackupPath              string     `gorm:"size:500" json:"backup_path"`
	LastBackupAt            *time.Time `json:"last_backup_at"`
}
