package db

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"my-pet/internal/model"
)

func TestMigratePetOwner(t *testing.T) {
	g, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.AutoMigrate(&model.Pet{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: "admin-id", Username: "admin", Role: "admin"}
	if err := g.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	// 存量数据：没有归属人
	legacy := &model.Pet{ID: "p1", Name: "豆豆"}
	if err := g.Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	owned := &model.Pet{ID: "p2", Name: "咪咪", OwnerID: "other-id"}
	if err := g.Create(owned).Error; err != nil {
		t.Fatal(err)
	}

	migratePetOwner(g)

	var got model.Pet
	g.First(&got, "id = ?", "p1")
	if got.OwnerID != admin.ID {
		t.Fatalf("legacy pet should belong to admin, got %q", got.OwnerID)
	}
	var got2 model.Pet
	g.First(&got2, "id = ?", "p2")
	if got2.OwnerID != "other-id" {
		t.Fatalf("owned pet should not be overwritten, got %q", got2.OwnerID)
	}
}
