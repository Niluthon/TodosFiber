package todo

import (
	"fmt"
	"time"
)

// dateLayout is the wire format for due dates (date only, no time component).
const dateLayout = "2006-01-02"

// ---------------------------------------------------------------------------
// HTTP request DTOs (bound from the request body/query and validated).
// ---------------------------------------------------------------------------

// CreateTodoRequest is the payload accepted by POST /tasks.
type CreateTodoRequest struct {
	Title   string     `json:"title" form:"title" validate:"required,max=200"`
	Status  TodoStatus `json:"status" form:"status" validate:"omitempty,oneof=pending done"`
	DueDate *string    `json:"due_date" form:"due_date"`
}

// UpdateTodoRequest is the payload accepted by PATCH /tasks/:id. Pointers are
// used so the handler can tell "field omitted" apart from "field set to zero".
type UpdateTodoRequest struct {
	Title   *string     `json:"title" form:"title" validate:"omitempty,min=1,max=200"`
	Status  *TodoStatus `json:"status" form:"status" validate:"omitempty,oneof=pending done"`
	DueDate *string     `json:"due_date" form:"due_date"`
}

// ListTodosQuery holds the supported query parameters for GET /tasks.
type ListTodosQuery struct {
	Status TodoStatus `query:"status" validate:"omitempty,oneof=pending done"`
	Page   int        `query:"page" validate:"omitempty,min=1"`
	Limit  int        `query:"limit" validate:"omitempty,min=1,max=100"`
}

// ---------------------------------------------------------------------------
// Service-layer DTOs (handler -> service).
// ---------------------------------------------------------------------------

// CreateTodoInput is the data the service needs to create a task.
type CreateTodoInput struct {
	Title   string     `validate:"notblank,max=200"`
	Status  TodoStatus `validate:"omitempty,oneof=pending done"`
	DueDate *time.Time
}

// UpdateTodoInput is the data the service needs to patch a task.
// A nil pointer means "leave unchanged"; ClearDueDate explicitly removes the date.
type UpdateTodoInput struct {
	ID           uint
	Title        *string     `validate:"omitempty,notblank,max=200"`
	Status       *TodoStatus `validate:"omitempty,oneof=pending done"`
	DueDate      *time.Time
	ClearDueDate bool
}

// ListFilter carries list options from the handler into the service/repository.
type ListFilter struct {
	Status TodoStatus
	Page   int
	Limit  int
}

// ---------------------------------------------------------------------------
// Persistence DTO (service <-> repository).
// ---------------------------------------------------------------------------

// TodoDto is the persistence-agnostic representation of a todo exchanged
// between the service and repository layers. Unlike the Todo GORM model, it has
// no database tags and no ORM hooks, so the service never has to know how (or
// where) a todo is stored. The repository owns translating it to and from the
// ORM model.
type TodoDto struct {
	ID        uint
	Title     string
	Status    TodoStatus
	DueDate   *time.Time
	CreatedAt time.Time
}

// ---------------------------------------------------------------------------
// Response DTO (service -> handler -> client).
// ---------------------------------------------------------------------------

// TodoResponse is the JSON representation of a task returned to the client.
type TodoResponse struct {
	ID        uint       `json:"id"`
	Title     string     `json:"title"`
	Status    TodoStatus `json:"status"`
	DueDate   *string    `json:"due_date"`
	CreatedAt time.Time  `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Mappers.
// ---------------------------------------------------------------------------

func parseDueDate(value string) (*time.Time, error) {
	t, err := time.Parse(dateLayout, value)
	if err != nil {
		return nil, fmt.Errorf("%w: due_date must use the YYYY-MM-DD format", ErrValidation)
	}
	return &t, nil
}

// ToInput converts a create request into a service input DTO.
func (r *CreateTodoRequest) ToInput() (CreateTodoInput, error) {
	input := CreateTodoInput{Title: r.Title, Status: r.Status}
	if r.DueDate != nil && *r.DueDate != "" {
		due, err := parseDueDate(*r.DueDate)
		if err != nil {
			return CreateTodoInput{}, err
		}
		input.DueDate = due
	}
	return input, nil
}

// ToInput converts a patch request into a service input DTO. An empty due_date
// string is treated as an explicit request to clear the date.
func (r *UpdateTodoRequest) ToInput(id uint) (UpdateTodoInput, error) {
	input := UpdateTodoInput{ID: id, Title: r.Title, Status: r.Status}
	if r.DueDate != nil {
		if *r.DueDate == "" {
			input.ClearDueDate = true
		} else {
			due, err := parseDueDate(*r.DueDate)
			if err != nil {
				return UpdateTodoInput{}, err
			}
			input.DueDate = due
		}
	}
	return input, nil
}

// ToFilter converts validated query parameters into a list filter, applying
// sane defaults for pagination.
func (q *ListTodosQuery) ToFilter() ListFilter {
	filter := ListFilter{Status: q.Status, Page: q.Page, Limit: q.Limit}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	return filter
}

// toTodoResponse maps a persistence DTO into the API response DTO.
func toTodoResponse(t TodoDto) TodoResponse {
	resp := TodoResponse{
		ID:        t.ID,
		Title:     t.Title,
		Status:    t.Status,
		CreatedAt: t.CreatedAt,
	}
	if t.DueDate != nil {
		formatted := t.DueDate.Format(dateLayout)
		resp.DueDate = &formatted
	}
	return resp
}

func toTodoResponses(todos []TodoDto) []TodoResponse {
	responses := make([]TodoResponse, 0, len(todos))
	for i := range todos {
		responses = append(responses, toTodoResponse(todos[i]))
	}
	return responses
}
