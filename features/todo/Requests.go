package todo

// CreateTodoRequest is the payload accepted by POST /tasks and the input the
// service consumes. DueDate is a date-only string on the wire ("2006-01-02");
// the service converts it to a time.Time before persisting.
type CreateTodoRequest struct {
	Title   string     `json:"title" form:"title" validate:"required,notblank,max=200"`
	Status  TodoStatus `json:"status" form:"status" validate:"omitempty,oneof=pending done"`
	DueDate *string    `json:"due_date" form:"due_date"`
}

// UpdateTodoRequest is the payload accepted by PATCH /tasks/:id and the input
// the service consumes. Pointer fields let the service tell "omitted" apart
// from "explicitly set"; an empty due_date string clears the date.
type UpdateTodoRequest struct {
	// ID is populated by the handler from the :id route parameter, never from
	// the body, so it is skipped during binding and validated by parseID.
	ID      uint        `json:"-" form:"-" validate:"-"`
	Title   *string     `json:"title" form:"title" validate:"omitempty,notblank,max=200"`
	Status  *TodoStatus `json:"status" form:"status" validate:"omitempty,oneof=pending done"`
	DueDate *string     `json:"due_date" form:"due_date"`
}

// ListTodosQuery carries list options from the handler into the service and
// repository. It is also bound directly from the request query string.
type ListTodosQuery struct {
	Status TodoStatus `query:"status" form:"status" validate:"omitempty,oneof=pending done"`
	Page   int        `query:"page" form:"page" validate:"omitempty,min=1"`
	Limit  int        `query:"limit" form:"limit" validate:"omitempty,min=1,max=100"`
}
