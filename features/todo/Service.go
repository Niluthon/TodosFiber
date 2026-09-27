package todo

import (
	"errors"
	"fmt"
	"strings"
)

// ErrValidation marks input that is well-formed but breaks a business rule.
// Callers should test for it with errors.Is; the HTTP layer maps it to 400.
var ErrValidation = errors.New("validation error")

type TodoServiceInterface interface {
	Create(input CreateTodoInput) (TodoResponse, error)
	Update(input UpdateTodoInput) (TodoResponse, error)
	Delete(id uint) error
	Get(id uint) (TodoResponse, error)
	List(filter ListFilter) ([]TodoResponse, error)
}

// TodoService implements the TodoServiceInterface port using the todo business rules. It
// has no knowledge of HTTP or of the concrete storage engine.
type TodoService struct {
	repo TodoRepositoryInterface
}

// TodoService must satisfy the TodoServiceInterface interface the handler depends on. This
// is an explicit compile-time check: if a method signature ever drifts, the
// package stops building here instead of failing later at the call site.
var _ TodoServiceInterface = (*TodoService)(nil)

// NewService returns a TodoServiceInterface that persists through repo.
func NewService(repo TodoRepositoryInterface) TodoServiceInterface {
	return &TodoService{repo: repo}
}

// Create validates input and stores a new todo.
func (s *TodoService) Create(input CreateTodoInput) (TodoResponse, error) {
	if strings.TrimSpace(input.Title) == "" {
		return TodoResponse{}, fmt.Errorf("%w: title is required", ErrValidation)
	}

	status := input.Status
	if status == "" {
		status = Pending
	}
	if err := ensureDoneHasTitle(input.Title, status); err != nil {
		return TodoResponse{}, err
	}

	t := &Todo{
		Title:   input.Title,
		Status:  status,
		DueDate: input.DueDate,
	}
	if err := s.repo.Create(t); err != nil {
		return TodoResponse{}, err
	}
	return toTodoResponse(t), nil
}

// Update applies a partial update to an existing todo.
func (s *TodoService) Update(input UpdateTodoInput) (TodoResponse, error) {
	t, err := s.repo.GetByID(input.ID)
	if err != nil {
		return TodoResponse{}, err
	}

	// Only fields that were actually sent are touched.
	if input.Title != nil {
		if strings.TrimSpace(*input.Title) == "" {
			return TodoResponse{}, fmt.Errorf("%w: title cannot be empty", ErrValidation)
		}
		t.Title = *input.Title
	}
	if input.Status != nil {
		if err := ensureDoneHasTitle(t.Title, *input.Status); err != nil {
			return TodoResponse{}, err
		}
		t.Status = *input.Status
	}
	switch {
	case input.ClearDueDate:
		t.DueDate = nil
	case input.DueDate != nil:
		t.DueDate = input.DueDate
	}

	if err := s.repo.Update(t); err != nil {
		return TodoResponse{}, err
	}
	return toTodoResponse(t), nil
}

// Delete removes a todo by id.
func (s *TodoService) Delete(id uint) error {
	return s.repo.Delete(id)
}

// Get returns a single todo by id.
func (s *TodoService) Get(id uint) (TodoResponse, error) {
	t, err := s.repo.GetByID(id)
	if err != nil {
		return TodoResponse{}, err
	}
	return toTodoResponse(t), nil
}

// List returns the todos that match filter.
func (s *TodoService) List(filter ListFilter) ([]TodoResponse, error) {
	todos, err := s.repo.List(filter)
	if err != nil {
		return nil, err
	}
	return toTodoResponses(todos), nil
}

// ensureDoneHasTitle enforces the rule that a task cannot be marked "done"
// without a title.
func ensureDoneHasTitle(title string, status TodoStatus) error {
	if status == Done && strings.TrimSpace(title) == "" {
		return fmt.Errorf("%w: a task cannot be marked as done without a title", ErrValidation)
	}
	return nil
}
