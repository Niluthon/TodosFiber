package todo

import (
	todo2 "TodosFiber/features/todo"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testValidator struct {
	validator *validator.Validate
}

func (v *testValidator) Validate(out interface{}) error { return v.validator.Struct(out) }

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "handler.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&todo2.TodoGorm{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	closeDB(t, db)

	validate, err := todo2.NewValidator()
	if err != nil {
		t.Fatalf("build validator: %v", err)
	}

	handler := todo2.NewTodoHandler(todo2.NewService(todo2.NewTodoRepository(db), validate))
	app := fiber.New(fiber.Config{
		StructValidator: &testValidator{validator: validate},
	})

	tasks := app.Group("/tasks")
	tasks.Post("/", handler.Create)
	tasks.Get("/", handler.List)
	tasks.Get("/:id", handler.Get)
	tasks.Patch("/:id", handler.Update)
	tasks.Delete("/:id", handler.Delete)

	return app
}

func doJSON(t *testing.T, app *fiber.App, method, path, body string) *http.Response {
	t.Helper()

	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	return resp
}

// closeDB registers a cleanup that closes the underlying sql.DB so Windows can
// delete the temporary database file when the test finishes.
func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
}

func TestHandlerCRUDFlow(t *testing.T) {
	app := newTestApp(t)

	// Create -> 201.
	resp := doJSON(t, app, http.MethodPost, "/tasks", `{"title":"Buy groceries","due_date":"2026-05-15"}`)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("POST /tasks status = %d, want 201", resp.StatusCode)
	}
	var created todo2.TodoResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == 0 || created.Status != todo2.Pending {
		t.Fatalf("unexpected created task: %+v", created)
	}

	// Get -> 200.
	resp = doJSON(t, app, http.MethodGet, "/tasks/1", "")
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("GET /tasks/1 status = %d, want 200", resp.StatusCode)
	}

	// List -> 200.
	resp = doJSON(t, app, http.MethodGet, "/tasks", "")
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("GET /tasks status = %d, want 200", resp.StatusCode)
	}

	// Patch -> 200.
	resp = doJSON(t, app, http.MethodPatch, "/tasks/1", `{"status":"done"}`)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("PATCH /tasks/1 status = %d, want 200", resp.StatusCode)
	}

	// Delete -> 204, then Get -> 404.
	resp = doJSON(t, app, http.MethodDelete, "/tasks/1", "")
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("DELETE /tasks/1 status = %d, want 204", resp.StatusCode)
	}
	resp = doJSON(t, app, http.MethodGet, "/tasks/1", "")
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("GET deleted task status = %d, want 404", resp.StatusCode)
	}
}

func TestHandlerRejectsBadInput(t *testing.T) {
	app := newTestApp(t)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"missing title", http.MethodPost, "/tasks", `{"status":"pending"}`, fiber.StatusBadRequest},
		{"invalid status", http.MethodPost, "/tasks", `{"title":"x","status":"nope"}`, fiber.StatusBadRequest},
		{"malformed json", http.MethodPost, "/tasks", `{"title":`, fiber.StatusBadRequest},
		{"bad id", http.MethodGet, "/tasks/abc", "", fiber.StatusBadRequest},
		{"missing task", http.MethodGet, "/tasks/42", "", fiber.StatusNotFound},
		{"update missing task", http.MethodPatch, "/tasks/1", `{"status":"done"}`, fiber.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, app, tc.method, tc.path, tc.body)
			if resp.StatusCode != tc.want {
				t.Fatalf("%s %s status = %d, want %d", tc.method, tc.path, resp.StatusCode, tc.want)
			}
		})
	}
}

func TestHandlerDoneWithoutTitle(t *testing.T) {
	app := newTestApp(t)

	// An existing task with a title can be completed.
	resp := doJSON(t, app, http.MethodPost, "/tasks", `{"title":"Task"}`)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("POST /tasks status = %d, want 201", resp.StatusCode)
	}

	// Clearing the title while marking it done must be rejected with 400.
	resp = doJSON(t, app, http.MethodPatch, "/tasks/1", `{"title":"","status":"done"}`)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("PATCH invalid status = %d, want 400", resp.StatusCode)
	}
}
