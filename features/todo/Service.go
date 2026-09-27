package todo

import (
	"errors"
	"fmt"
	"strings"
)

// ErrValidation marks input that is well-formed but breaks a business rule.
// The handler turns it into an HTTP 400.
var ErrValidation = errors.New("validation error")

// ServiceInterface holds the business logic and is independent of HTTP/GORM.
type ServiceInterface interface {
	Create(input CreateTodoInput) (TodoResponse, error)
	Update(input UpdateTodoInput) (TodoResponse, error)
	Delete(id uint) error
	Get(id uint) (TodoResponse, error)
	List(filter ListFilter) ([]TodoResponse, error)
}

type service struct {
	repo RepositoryInterface
}

// NewTodoService builds the service on top of a repository dependency.
func NewTodoService(repo RepositoryInterface) ServiceInterface {
	return &service{repo: repo}
}

func (s *service) Create(input CreateTodoInput) (TodoResponse, error) {
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

func (s *service) Update(input UpdateTodoInput) (TodoResponse, error) {
	t, err := s.repo.GetByID(input.ID)
	if err != nil {
		return TodoResponse{}, err
	}

	// Apply the patch: only fields that were actually sent are touched.
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

func (s *service) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *service) Get(id uint) (TodoResponse, error) {
	t, err := s.repo.GetByID(id)
	if err != nil {
		return TodoResponse{}, err
	}
	return toTodoResponse(t), nil
}

func (s *service) List(filter ListFilter) ([]TodoResponse, error) {
	todos, err := s.repo.List(filter)
	if err != nil {
		return nil, err
	}
	return toTodoResponses(todos), nil
}

// ensureDoneHasTitle enforces the assignment rule: a task cannot be marked
// "done" if it has no title.
func ensureDoneHasTitle(title string, status TodoStatus) error {
	if status == Done && strings.TrimSpace(title) == "" {
		return fmt.Errorf("%w: a task cannot be marked as done without a title", ErrValidation)
	}
	return nil
}
