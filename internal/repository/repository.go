package repository

import (
	"context"
	"errors"
	"reflect"

	"gorm.io/gorm"
)

// Repository is a slim, domain-free generic data-access layer built on
// GORM's generics API (gorm.G[T]). It provides the small coherent CRUD
// surface the application actually uses.
type Repository[T any] struct {
	db *gorm.DB
}

// New creates a new generic Repository for model T.
func New[T any](db *gorm.DB) *Repository[T] {
	return &Repository[T]{db: db}
}

// DB returns the underlying *gorm.DB for advanced queries.
func (r *Repository[T]) DB() *gorm.DB {
	return r.db
}

// WithTx returns a new Repository that operates within the given transaction.
func (r *Repository[T]) WithTx(tx *gorm.DB) *Repository[T] {
	return &Repository[T]{db: tx}
}

// Create inserts a single record.
func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	return gorm.G[T](r.db).Create(ctx, entity)
}

// FindByID retrieves a single record by primary key.
func (r *Repository[T]) FindByID(ctx context.Context, id uint) (T, error) {
	return gorm.G[T](r.db).Where("id = ?", id).First(ctx)
}

// FindAll retrieves all records ordered by primary key.
func (r *Repository[T]) FindAll(ctx context.Context) ([]T, error) {
	return gorm.G[T](r.db).Order("id ASC").Find(ctx)
}

// Update applies non-zero struct fields of entity to the row with the same ID.
// The entity's ID field is read via reflection (supports gorm.Model embedding).
// Returns the number of affected rows.
func (r *Repository[T]) Update(ctx context.Context, entity T) (int64, error) {
	id := extractID(entity)
	if id == 0 {
		// No ID set — nothing to update; return 0 without error to preserve
		// the "missing id returns 0" semantics without hitting the DB.
		return 0, nil
	}
	n, err := gorm.G[T](r.db).Where("id = ?", id).Updates(ctx, entity)
	return int64(n), err
}

// Delete removes a record by id. It is implemented as load-then-delete so
// model hooks (BeforeDelete/AfterDelete) receive a populated primary key.
// Deleting a missing id returns 0 with no error.
func (r *Repository[T]) Delete(ctx context.Context, id uint) (int64, error) {
	entity, err := r.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	result := r.db.WithContext(ctx).Delete(&entity)
	return result.RowsAffected, result.Error
}

// Count returns the number of records.
func (r *Repository[T]) Count(ctx context.Context) (int64, error) {
	return gorm.G[T](r.db).Count(ctx, "id")
}

// ChildRepository extends Repository with parent-scoped queries. Parent rows
// are ordered by position ASC, the canonical trip ordering.
type ChildRepository[T any] struct {
	Repository[T]
	parentColumn string
}

// NewChild creates a new ChildRepository for model T with the given parent FK column.
func NewChild[T any](db *gorm.DB, parentColumn string) *ChildRepository[T] {
	return &ChildRepository[T]{
		Repository:   Repository[T]{db: db},
		parentColumn: parentColumn,
	}
}

// WithTx returns a new ChildRepository that operates within the given transaction.
// It returns *ChildRepository[T] so child methods survive rebinding.
func (c *ChildRepository[T]) WithTx(tx *gorm.DB) *ChildRepository[T] {
	return &ChildRepository[T]{
		Repository:   Repository[T]{db: tx},
		parentColumn: c.parentColumn,
	}
}

// ListByParent returns all rows belonging to parentID ordered by position ASC.
func (c *ChildRepository[T]) ListByParent(ctx context.Context, parentID uint) ([]T, error) {
	return gorm.G[T](c.db).Where(c.parentColumn+" = ?", parentID).Order("position ASC").Find(ctx)
}

// CountByParent counts rows belonging to parentID.
func (c *ChildRepository[T]) CountByParent(ctx context.Context, parentID uint) (int64, error) {
	return gorm.G[T](c.db).Where(c.parentColumn+" = ?", parentID).Count(ctx, "id")
}

// extractID reads the ID field from entity via reflection. It handles both
// direct ID fields and IDs promoted through embedded gorm.Model.
func extractID[T any](entity T) uint {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0
	}
	field := v.FieldByName("ID")
	if !field.IsValid() {
		return 0
	}
	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(field.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val := field.Int()
		if val < 0 {
			return 0
		}
		return uint(val)
	default:
		return 0
	}
}
