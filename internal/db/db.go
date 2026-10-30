package db

import (
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"my-pet/internal/model"
)

func Open(dbPath string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	g, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := g.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if err := g.AutoMigrate(&model.Pet{}, &model.FeedRecord{}, &model.Reminder{},
		&model.LogEntry{}, &model.User{}, &model.Setting{}); err != nil {
		return nil, err
	}
	seed(g)
	return g, nil
}

func seed(g *gorm.DB) {
	var count int64
	g.Model(&model.User{}).Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		g.Create(&model.User{Username: "admin", PasswordHash: string(hash), Role: "admin"})
	}
	var scount int64
	g.Model(&model.Setting{}).Count(&scount)
	if scount == 0 {
		g.Create(&model.Setting{ReminderIntervalMinutes: 1, BackupPath: "./data/backups"})
	}
}
