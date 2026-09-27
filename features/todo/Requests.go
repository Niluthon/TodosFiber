package todo

import "time"

type CreateTodoRequest struct {
	// Replacing `notblank` with `required`:
	Title   string     `validate:"required,max=255"`
	Status  TodoStatus `validate:"omitempty,oneof=pending in_progress completed"`
	DueDate *time.Time `validate:"omitempty"`
}

type UpdateTodoRequest struct {
	ID uint `validate:"required"`
	// For optional pointers, `omitempty` prevents validation if nil.
	// If non-nil, `required` enforces non-empty string value:
	Title        *string     `validate:"omitempty,required,max=255"`
	Status       *TodoStatus `validate:"omitempty,oneof=pending in_progress completed"`
	ClearDueDate bool        `validate:"-"`
	DueDate      *time.Time  `validate:"omitempty"`
}

// ListTodosQuery carries list options from the handler into the service/repository.
type ListTodosQuery struct {
	Status TodoStatus
	Page   int
	Limit  int
}
