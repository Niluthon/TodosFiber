package todo

import (
	"fmt"
	"strings"
)

// TodoServiceInterface is the business-logic port the HTTP handler depends on.
type TodoServiceInterface interface {
	Create(input CreateTodoRequest) (TodoResponse, error)
	Update(input UpdateTodoRequest) (TodoResponse, error)
	Delete(id uint) error
	Get(id uint) (TodoResponse, error)
	List(filter ListTodosQuery) ([]TodoResponse, error)
}

// TodoService implements TodoServiceInterface using the todo business rules. It
// has no knowledge of HTTP or of the concrete storage engine.
type TodoService struct {
	repo     TodoRepositoryInterface
	validate Validator
}

// TodoService must satisfy the TodoServiceInterface the handler depends on. This
// is an explicit compile-time check: if a method signature ever drifts, the
// package stops building here instead of failing later at the call site.
var _ TodoServiceInterface = (*TodoService)(nil)

// NewService returns a TodoServiceInterface that persists through repo and
// enforces the todo validation rules with validate. validate must be non-nil;
// use NewValidator.
func NewService(repo TodoRepositoryInterface, validate Validator) TodoServiceInterface {
	return &TodoService{repo: repo, validate: validate}
}

// Create validates the input and then stores a new todo.
func (s *TodoService) Create(input CreateTodoRequest) (TodoResponse, error) {
	if err := s.validate.Struct(input); err != nil {
		return TodoResponse{}, inputError(err)
	}

	status := input.Status
	if status == "" {
		status = Pending
	}
	if err := ensureDoneHasTitle(input.Title, status); err != nil {
		return TodoResponse{}, err
	}

	due, err := parseOptionalDueDate(input.DueDate)
	if err != nil {
		return TodoResponse{}, err
	}

	created, err := s.repo.Create(TodoDto{
		Title:   input.Title,
		Status:  status,
		DueDate: due,
	})
	if err != nil {
		return TodoResponse{}, err
	}
	return toTodoResponse(created), nil
}

// Update validates the input, applies the partial update to the existing todo
// and persists the result. Only fields that were actually sent are touched.
func (s *TodoService) Update(input UpdateTodoRequest) (TodoResponse, error) {
	if err := s.validate.Struct(input); err != nil {
		return TodoResponse{}, inputError(err)
	}

	dto, err := s.repo.GetByID(input.ID)
	if err != nil {
		return TodoResponse{}, err
	}

	if input.Title != nil {
		dto.Title = *input.Title
	}
	if input.Status != nil {
		dto.Status = *input.Status
	}
	if input.DueDate != nil {
		if *input.DueDate == "" {
			dto.DueDate = nil
		} else {
			due, err := parseDueDate(*input.DueDate)
			if err != nil {
				return TodoResponse{}, err
			}
			dto.DueDate = due
		}
	}

	if err := ensureDoneHasTitle(dto.Title, dto.Status); err != nil {
		return TodoResponse{}, err
	}

	updated, err := s.repo.Update(dto)
	if err != nil {
		return TodoResponse{}, err
	}
	return toTodoResponse(updated), nil
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

// List returns the todos that match filter, with pagination defaults applied.
func (s *TodoService) List(filter ListTodosQuery) ([]TodoResponse, error) {
	todos, err := s.repo.List(filter.ToFilter())
	if err != nil {
		return nil, err
	}
	return toTodoList(todos), nil
}

// ensureDoneHasTitle enforces the rule that a task cannot be marked "done"
// without a title.
func ensureDoneHasTitle(title string, status TodoStatus) error {
	if status == Done && strings.TrimSpace(title) == "" {
		return fmt.Errorf("%w: a task cannot be marked as done without a title", ErrValidation)
	}
	return nil
}
