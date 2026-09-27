package todo

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNotFound is returned by the repository when a task does not exist.
// The service/handler translate it into an HTTP 404.
var ErrNotFound = errors.New("todo not found")

// RepositoryInterface is the data-access contract. It only knows about the
// Todo entity, never about DTOs or HTTP concerns.
type RepositoryInterface interface {
	Create(todo *Todo) error
	Update(todo *Todo) error
	Delete(id uint) error
	GetByID(id uint) (*Todo, error)
	List(filter ListFilter) ([]Todo, error)
}

// repository is the GORM-backed implementation. The database handle is
// injected through NewTodoRepository so the data store is easy to swap/test.
type repository struct {
	db *gorm.DB
}

// NewTodoRepository builds a repository backed by the given database instance.
func NewTodoRepository(db *gorm.DB) RepositoryInterface {
	return &repository{db: db}
}

func (r *repository) Create(t *Todo) error {
	if err := r.db.Create(t).Error; err != nil {
		return fmt.Errorf("repository: create todo: %w", err)
	}
	return nil
}

func (r *repository) Update(t *Todo) error {
	if err := r.db.Save(t).Error; err != nil {
		return fmt.Errorf("repository: update todo: %w", err)
	}
	return nil
}

func (r *repository) Delete(id uint) error {
	result := r.db.Delete(&Todo{}, id)
	if result.Error != nil {
		return fmt.Errorf("repository: delete todo: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repository) GetByID(id uint) (*Todo, error) {
	var t Todo
	if err := r.db.First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository: get todo: %w", err)
	}
	return &t, nil
}

// List returns tasks newest first, optionally filtered by status and paginated.
func (r *repository) List(filter ListFilter) ([]Todo, error) {
	query := r.db.Model(&Todo{}).Order("created_at DESC, id DESC")

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		query = query.Offset(offset).Limit(filter.Limit)
	}

	var todos []Todo
	if err := query.Find(&todos).Error; err != nil {
		return nil, fmt.Errorf("repository: list todos: %w", err)
	}
	return todos, nil
}
