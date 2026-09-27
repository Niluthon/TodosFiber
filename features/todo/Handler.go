package todo

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// TodoHandler adapts HTTP requests to TodoServiceInterface calls. It owns only transport
// concerns: binding, validation and mapping errors to status codes.
type TodoHandler struct {
	service TodoServiceInterface
}

// NewTodoHandler returns a TodoHandler that delegates to the TodoServiceInterface port.
func NewTodoHandler(service TodoServiceInterface) *TodoHandler {
	return &TodoHandler{service: service}
}

// Create handles POST /tasks.
func (h *TodoHandler) Create(c fiber.Ctx) error {
	req := new(CreateTodoRequest)
	if err := c.Bind().Body(req); err != nil {
		return respondBadRequest(c, "invalid request body", err)
	}

	input, err := req.ToInput()
	if err != nil {
		return respondError(c, err)
	}

	resp, err := h.service.Create(input)
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// List handles GET /tasks (supports ?status=, ?page= and ?limit=).
func (h *TodoHandler) List(c fiber.Ctx) error {
	query := new(ListTodosQuery)
	if err := c.Bind().Query(query); err != nil {
		return respondBadRequest(c, "invalid query parameters", err)
	}

	todos, err := h.service.List(query.ToFilter())
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(todos)
}

// Get handles GET /tasks/:id.
func (h *TodoHandler) Get(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, err)
	}

	resp, err := h.service.Get(id)
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// Update handles PATCH /tasks/:id.
func (h *TodoHandler) Update(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, err)
	}

	req := new(UpdateTodoRequest)
	if err := c.Bind().Body(req); err != nil {
		return respondBadRequest(c, "invalid request body", err)
	}

	input, err := req.ToInput(id)
	if err != nil {
		return respondError(c, err)
	}

	resp, err := h.service.Update(input)
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// Delete handles DELETE /tasks/:id.
func (h *TodoHandler) Delete(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, err)
	}

	if err := h.service.Delete(id); err != nil {
		return respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// parseID reads and validates the :id route parameter.
func parseID(c fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: id must be a positive integer", ErrValidation)
	}
	return uint(id), nil
}

// respondError maps sentinel errors from the service/repository onto HTTP
// status codes. Unknown errors never leak internal details to the client.
func respondError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrValidation):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

func respondBadRequest(c fiber.Ctx, message string, err error) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":   message,
		"details": err.Error(),
	})
}
