package todo

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNotFound is returned when a task does not exist.
// Callers should test for it with errors.Is.
var ErrNotFound = errors.New("todo not found")

type TodoRepositoryInterface interface {
	Create(todo *Todo) error
	Update(todo *Todo) error
	Delete(id uint) error
	GetByID(id uint) (*Todo, error)
	List(filter ListFilter) ([]Todo, error)
}

// TodoRepository is the GORM-backed adapter for the TodoRepositoryInterface port. The
// database handle is injected through NewTodoRepository, keeping the storage
// engine replaceable and the type easy to fake in tests.
type TodoRepository struct {
	db *gorm.DB
}

// TodoRepository must satisfy the TodoRepositoryInterface interface the service depends on.
// This is an explicit compile-time check: if a method signature ever drifts,
// the package stops building here instead of failing later at the call site.
var _ TodoRepositoryInterface = (*TodoRepository)(nil)

// NewTodoRepository returns a TodoRepository backed by db.
func NewTodoRepository(db *gorm.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

// Create inserts t and fills in its generated fields (ID, CreatedAt).
func (r *TodoRepository) Create(t *Todo) error {
	if err := r.db.Create(t).Error; err != nil {
		return fmt.Errorf("create todo: %w", err)
	}
	return nil
}

// Update persists every field of t.
func (r *TodoRepository) Update(t *Todo) error {
	if err := r.db.Save(t).Error; err != nil {
		return fmt.Errorf("update todo %d: %w", t.ID, err)
	}
	return nil
}

// Delete removes the todo with the given id. It returns ErrNotFound when no
// row was affected.
func (r *TodoRepository) Delete(id uint) error {
	result := r.db.Delete(&Todo{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete todo %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// GetByID returns the todo with the given id, or ErrNotFound.
func (r *TodoRepository) GetByID(id uint) (*Todo, error) {
	var t Todo
	if err := r.db.First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get todo %d: %w", id, err)
	}
	return &t, nil
}

// List returns todos newest first, optionally filtered by status and paginated.
// A filter with Limit <= 0 returns every match.
func (r *TodoRepository) List(filter ListFilter) ([]Todo, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}

	query := r.db.Model(&Todo{}).
		Order("created_at DESC, id DESC")

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Limit > 0 {
		query = query.Offset((filter.Page - 1) * filter.Limit).Limit(filter.Limit)
	}

	var todos []Todo
	if err := query.Find(&todos).Error; err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	return todos, nil
}
