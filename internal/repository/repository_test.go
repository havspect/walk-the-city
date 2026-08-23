package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testModel is a domain-free model for repository tests.
type testModel struct {
	gorm.Model
	Name     string `gorm:"size:255"`
	Position int
}

// childModel has a parent foreign key for ChildRepository tests.
type childModel struct {
	gorm.Model
	ParentID uint `gorm:"column:parent_id;index"`
	Name     string
	Position int `gorm:"not null"`
}

// hookModel verifies load-then-delete hook semantics.
type hookModel struct {
	gorm.Model
	Name      string
	DeletedID uint `gorm:"-"`
}

func (h *hookModel) AfterDelete(tx *gorm.DB) error {
	h.DeletedID = h.ID
	if h.ID == 0 {
		return errors.New("hook received zero ID")
	}
	return nil
}

func setupRepoDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:memrepo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		t.Fatalf("pragma: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestRepository_CreateAndFindByID(t *testing.T) {
	db := setupRepoDB(t, &testModel{})
	repo := New[testModel](db)
	ctx := context.Background()

	m := &testModel{Name: "alpha"}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	if m.ID == 0 {
		t.Fatalf("expected ID assigned")
	}
	found, err := repo.FindByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("findByID: %v", err)
	}
	if found.Name != "alpha" {
		t.Errorf("expected alpha, got %q", found.Name)
	}
}

func TestRepository_FindByID_Missing(t *testing.T) {
	db := setupRepoDB(t, &testModel{})
	repo := New[testModel](db)
	ctx := context.Background()
	_, err := repo.FindByID(ctx, 99999)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestRepository_FindAll(t *testing.T) {
	db := setupRepoDB(t, &testModel{})
	repo := New[testModel](db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := repo.Create(ctx, &testModel{Name: fmt.Sprintf("n%d", i)}); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("findAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}
	// Ordered by id ASC
	for i := 1; i < len(all); i++ {
		if all[i].ID < all[i-1].ID {
			t.Errorf("not ordered by id ASC")
		}
	}
}

func TestRepository_Update(t *testing.T) {
	db := setupRepoDB(t, &testModel{})
	repo := New[testModel](db)
	ctx := context.Background()

	m := &testModel{Name: "before", Position: 1}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Update non-zero field
	updated := testModel{Model: gorm.Model{ID: m.ID}, Name: "after"}
	n, err := repo.Update(ctx, updated)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 affected, got %d", n)
	}
	found, _ := repo.FindByID(ctx, m.ID)
	if found.Name != "after" {
		t.Errorf("expected after, got %q", found.Name)
	}
	if found.Position != 1 {
		t.Errorf("position should remain 1 (zero value not updated), got %d", found.Position)
	}
	// Updating missing id returns 0 with no error
	n, err = repo.Update(ctx, testModel{Model: gorm.Model{ID: 99999}, Name: "ghost"})
	if err != nil {
		t.Fatalf("update missing: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 affected for missing id, got %d", n)
	}
}

func TestRepository_Delete(t *testing.T) {
	db := setupRepoDB(t, &testModel{})
	repo := New[testModel](db)
	ctx := context.Background()

	m := &testModel{Name: "todel"}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	n, err := repo.Delete(ctx, m.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
	_, err = repo.FindByID(ctx, m.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected not found after delete, got %v", err)
	}
	// Deleting missing id returns 0 with no error
	n, err = repo.Delete(ctx, 99999)
	if err != nil {
		t.Fatalf("delete missing: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 for missing delete, got %d", n)
	}
}

func TestRepository_Delete_LoadThenDelete_Hook(t *testing.T) {
	db := setupRepoDB(t, &hookModel{})
	repo := New[hookModel](db)
	ctx := context.Background()

	m := &hookModel{Name: "hooktest"}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	n, err := repo.Delete(ctx, m.ID)
	if err != nil {
		t.Fatalf("delete with hook: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
	// AfterDelete should have seen populated ID — verified by no error.
	// Also verify soft-delete: unscoped find should still find it
	var found hookModel
	if err := db.Unscoped().First(&found, m.ID).Error; err != nil {
		t.Fatalf("unscoped find after soft delete: %v", err)
	}
	if found.DeletedAt.Time.IsZero() {
		t.Errorf("expected soft deleted")
	}
}

func TestRepository_Count(t *testing.T) {
	db := setupRepoDB(t, &testModel{})
	repo := New[testModel](db)
	ctx := context.Background()

	c, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("count empty: %v", err)
	}
	if c != 0 {
		t.Errorf("expected 0, got %d", c)
	}
	for i := 0; i < 3; i++ {
		_ = repo.Create(ctx, &testModel{Name: fmt.Sprintf("c%d", i)})
	}
	c, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("count populated: %v", err)
	}
	if c != 3 {
		t.Errorf("expected 3, got %d", c)
	}
}

func TestChildRepository_ListAndCountByParent(t *testing.T) {
	db := setupRepoDB(t, &childModel{})
	childRepo := NewChild[childModel](db, "parent_id")
	ctx := context.Background()

	// Parent 1: positions 2,0,1 scrambled
	for i, pos := range []int{2, 0, 1} {
		if err := childRepo.Create(ctx, &childModel{ParentID: 1, Name: fmt.Sprintf("p1-%d", i), Position: pos}); err != nil {
			t.Fatalf("create p1: %v", err)
		}
	}
	// Parent 2: single
	if err := childRepo.Create(ctx, &childModel{ParentID: 2, Name: "p2", Position: 0}); err != nil {
		t.Fatalf("create p2: %v", err)
	}

	list, err := childRepo.ListByParent(ctx, 1)
	if err != nil {
		t.Fatalf("listByParent: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 for parent 1, got %d", len(list))
	}
	for i := 1; i < len(list); i++ {
		if list[i].Position < list[i-1].Position {
			t.Errorf("not ordered by position ASC")
		}
	}
	if list[0].Position != 0 || list[1].Position != 1 || list[2].Position != 2 {
		t.Errorf("unexpected position order %v", list)
	}

	c1, _ := childRepo.CountByParent(ctx, 1)
	if c1 != 3 {
		t.Errorf("count parent 1 expected 3, got %d", c1)
	}
	c2, _ := childRepo.CountByParent(ctx, 2)
	if c2 != 1 {
		t.Errorf("count parent 2 expected 1, got %d", c2)
	}
	empty, _ := childRepo.ListByParent(ctx, 999)
	if len(empty) != 0 {
		t.Errorf("expected empty for unknown parent, got %d", len(empty))
	}
}

func TestRepository_WithTx_CommitAndRollback(t *testing.T) {
	db := setupRepoDB(t, &testModel{})
	repo := New[testModel](db)
	ctx := context.Background()

	// Commit
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := repo.WithTx(tx)
		if err := txRepo.Create(ctx, &testModel{Name: "in-tx"}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("tx commit: %v", err)
	}
	all, _ := repo.FindAll(ctx)
	if len(all) != 1 {
		t.Errorf("expected 1 after commit, got %d", len(all))
	}

	// Rollback via injected error
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := repo.WithTx(tx)
		if err := txRepo.Create(ctx, &testModel{Name: "rollback"}); err != nil {
			return err
		}
		return errors.New("injected failure")
	})
	if err == nil {
		t.Fatalf("expected injected error")
	}
	all, _ = repo.FindAll(ctx)
	if len(all) != 1 {
		t.Errorf("expected 1 after rollback (second insert rolled back), got %d", len(all))
	}
	// Verify via unscoped count still 1
	c, _ := repo.Count(ctx)
	if c != 1 {
		t.Errorf("count expected 1 after rollback, got %d", c)
	}
}

func TestChildRepository_WithTx(t *testing.T) {
	db := setupRepoDB(t, &childModel{})
	childRepo := NewChild[childModel](db, "parent_id")
	ctx := context.Background()

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := childRepo.WithTx(tx)
		for i := 0; i < 2; i++ {
			if err := txRepo.Create(ctx, &childModel{ParentID: 10, Name: fmt.Sprintf("tx-%d", i), Position: i}); err != nil {
				return err
			}
		}
		// Child methods must survive rebinding
		list, err := txRepo.ListByParent(ctx, 10)
		if err != nil {
			return err
		}
		if len(list) != 2 {
			return fmt.Errorf("expected 2 in tx, got %d", len(list))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("child tx: %v", err)
	}
	list, _ := childRepo.ListByParent(ctx, 10)
	if len(list) != 2 {
		t.Errorf("expected 2 after commit, got %d", len(list))
	}
}

func TestRepository_DB_EscapeHatch(t *testing.T) {
	db := setupRepoDB(t, &testModel{})
	repo := New[testModel](db)
	ctx := context.Background()
	_ = repo.Create(ctx, &testModel{Name: "escape"})
	// Use DB() for advanced query
	var count int64
	if err := repo.DB().WithContext(ctx).Model(&testModel{}).Where("name = ?", "escape").Count(&count).Error; err != nil {
		t.Fatalf("escape hatch: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 via escape hatch, got %d", count)
	}
}
