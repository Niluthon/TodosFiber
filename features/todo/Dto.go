package todo

import (
	"fmt"
	"time"
)

// dateLayout is the wire format for due dates (date only, no time component).
const dateLayout = "2006-01-02"

// ---------------------------------------------------------------------------
// Persistence DTO (service <-> repository).
// ---------------------------------------------------------------------------

// TodoDto is the persistence-agnostic representation of a todo exchanged
// between the service and repository layers. Unlike the TodoGorm GORM model, it has
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

// parseDueDate parses a date-only wire value. Failures are wrapped in
// ErrValidation so the HTTP layer maps them to 400.
func parseDueDate(value string) (*time.Time, error) {
	t, err := time.Parse(dateLayout, value)
	if err != nil {
		return nil, fmt.Errorf("%w: due_date must use the YYYY-MM-DD format", ErrValidation)
	}
	return &t, nil
}

// parseOptionalDueDate parses an optional date-only wire value. A nil or empty
// value means "no due date".
func parseOptionalDueDate(value *string) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	return parseDueDate(*value)
}

// toModel maps the persistence DTO onto the GORM model. It is the only place
// that couples the domain representation to the ORM type.
func toModel(dto TodoDto) *TodoGorm {
	return &TodoGorm{
		ID:        dto.ID,
		Title:     dto.Title,
		Status:    dto.Status,
		DueDate:   dto.DueDate,
		CreatedAt: dto.CreatedAt,
	}
}

// toDto maps a GORM model back into the persistence DTO.
func toDto(model *TodoGorm) TodoDto {
	return TodoDto{
		ID:        model.ID,
		Title:     model.Title,
		Status:    model.Status,
		DueDate:   model.DueDate,
		CreatedAt: model.CreatedAt,
	}
}

// ToFilter applies sane defaults for pagination.
func (q *ListTodosQuery) ToFilter() ListTodosQuery {
	filter := ListTodosQuery{Status: q.Status, Page: q.Page, Limit: q.Limit}
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

// toTodoList maps a slice of persistence DTOs into response DTOs.
func toTodoList(todos []TodoDto) []TodoResponse {
	responses := make([]TodoResponse, 0, len(todos))
	for i := range todos {
		responses = append(responses, toTodoResponse(todos[i]))
	}
	return responses
}
