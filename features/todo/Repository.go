package todo

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNotFound is returned when a task does not exist.
// Callers should test for it with errors.Is.
var ErrNotFound = errors.New("todo not found")

// TodoRepositoryInterface is the storage port the service depends on. It speaks
// exclusively in persistence-agnostic TodoDto values so the service never sees
// the GORM model. The concrete adapter is responsible for all ORM mapping.
type TodoRepositoryInterface interface {
	Create(todo TodoDto) (TodoDto, error)
	Update(todo TodoDto) (TodoDto, error)
	Delete(id uint) error
	GetByID(id uint) (TodoDto, error)
	List(filter ListTodosQuery) ([]TodoDto, error)
}

// TodoRepository is the GORM-backed adapter for the TodoRepositoryInterface port. The
// database handle is injected through NewTodoRepository, keeping the storage
// engine replaceable and the type easy to fake in tests. It is also the only
// place in the package that knows about the TodoGorm ORM model.
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

// Create inserts a new todo built from dto and returns it with the generated
// fields (ID, CreatedAt, and any database defaults) populated.
func (r *TodoRepository) Create(dto TodoDto) (TodoDto, error) {
	model := toModel(dto)
	if err := r.db.Create(model).Error; err != nil {
		return TodoDto{}, fmt.Errorf("create todo: %w", err)
	}
	return toDto(model), nil
}

// Update persists every field of the todo built from dto and returns the stored
// representation.
func (r *TodoRepository) Update(dto TodoDto) (TodoDto, error) {
	model := toModel(dto)
	if err := r.db.Save(model).Error; err != nil {
		return TodoDto{}, fmt.Errorf("update todo %d: %w", dto.ID, err)
	}
	return toDto(model), nil
}

// Delete removes the todo with the given id. It returns ErrNotFound when no
// row was affected.
func (r *TodoRepository) Delete(id uint) error {
	result := r.db.Delete(&TodoGorm{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete todo %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// GetByID returns the todo with the given id as a DTO, or ErrNotFound.
func (r *TodoRepository) GetByID(id uint) (TodoDto, error) {
	var model TodoGorm
	if err := r.db.First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TodoDto{}, ErrNotFound
		}
		return TodoDto{}, fmt.Errorf("get todo %d: %w", id, err)
	}
	return toDto(&model), nil
}

// List returns todos newest first, optionally filtered by status and paginated.
// A filter with Limit <= 0 returns every match.
func (r *TodoRepository) List(filter ListTodosQuery) ([]TodoDto, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}

	query := r.db.Model(&TodoGorm{}).
		Order("created_at DESC, id DESC")

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Limit > 0 {
		query = query.Offset((filter.Page - 1) * filter.Limit).Limit(filter.Limit)
	}

	var models []TodoGorm
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}

	dtos := make([]TodoDto, 0, len(models))
	for i := range models {
		dtos = append(dtos, toDto(&models[i]))
	}
	return dtos, nil
}
