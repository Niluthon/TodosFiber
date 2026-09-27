package todo

import (
	todo2 "TodosFiber/features/todo"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestService builds a service backed by a throwaway SQLite file.
func newTestService(t *testing.T) todo2.TodoServiceInterface {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "test.db")
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
	return todo2.NewService(todo2.NewTodoRepository(db), validate)
}

func TestCreateDefaultsToPending(t *testing.T) {
	svc := newTestService(t)

	got, err := svc.Create(todo2.CreateTodoRequest{Title: "Buy groceries"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if got.Status != todo2.Pending {
		t.Errorf("status = %q, want %q", got.Status, todo2.Pending)
	}
	if got.ID == 0 {
		t.Error("expected a non-zero ID")
	}
}

func TestCreateDoneWithoutTitleIsRejected(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Create(todo2.CreateTodoRequest{Title: "   ", Status: todo2.Done})
	if !errors.Is(err, todo2.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestUpdateDoneWithoutTitleIsRejected(t *testing.T) {
	svc := newTestService(t)

	created, err := svc.Create(todo2.CreateTodoRequest{Title: "Task"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	empty := ""
	done := todo2.Done
	_, err = svc.Update(todo2.UpdateTodoRequest{ID: created.ID, Title: &empty, Status: &done})
	if !errors.Is(err, todo2.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestUpdateNotFound(t *testing.T) {
	svc := newTestService(t)

	title := "Nope"
	_, err := svc.Update(todo2.UpdateTodoRequest{ID: 999, Title: &title})
	if !errors.Is(err, todo2.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	svc := newTestService(t)

	if err := svc.Delete(12345); !errors.Is(err, todo2.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListIsNewestFirstAndFilterable(t *testing.T) {
	svc := newTestService(t)

	for _, title := range []string{"first", "second", "third"} {
		if _, err := svc.Create(todo2.CreateTodoRequest{Title: title}); err != nil {
			t.Fatalf("Create(%q) returned error: %v", title, err)
		}
	}
	if _, err := svc.Create(todo2.CreateTodoRequest{Title: "finished", Status: todo2.Done}); err != nil {
		t.Fatalf("Create(done) returned error: %v", err)
	}

	all, err := svc.List(todo2.ListTodosQuery{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("len(all) = %d, want 4", len(all))
	}
	if all[0].Title != "finished" {
		t.Errorf("first item = %q, want the newest task %q", all[0].Title, "finished")
	}

	filtered, err := svc.List(todo2.ListTodosQuery{Status: todo2.Done, Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List(done) returned error: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Title != "finished" {
		t.Fatalf("filtered = %+v, want only the finished task", filtered)
	}
}

// fakeRepository is a minimal in-memory TodoRepositoryInterface. It proves the TodoServiceInterface can
// be unit tested without touching a database, which is the point of the port.
type fakeRepository struct {
	todos  map[uint]todo2.TodoDto
	nextID uint
}

var _ todo2.TodoRepositoryInterface = (*fakeRepository)(nil)

func newFakeRepository() *fakeRepository {
	return &fakeRepository{todos: make(map[uint]todo2.TodoDto)}
}

func (f *fakeRepository) Create(t todo2.TodoDto) (todo2.TodoDto, error) {
	f.nextID++
	t.ID = f.nextID
	f.todos[t.ID] = t
	return t, nil
}

func (f *fakeRepository) Update(t todo2.TodoDto) (todo2.TodoDto, error) {
	if _, ok := f.todos[t.ID]; !ok {
		return todo2.TodoDto{}, todo2.ErrNotFound
	}
	f.todos[t.ID] = t
	return t, nil
}

func (f *fakeRepository) Delete(id uint) error {
	if _, ok := f.todos[id]; !ok {
		return todo2.ErrNotFound
	}
	delete(f.todos, id)
	return nil
}

func (f *fakeRepository) GetByID(id uint) (todo2.TodoDto, error) {
	t, ok := f.todos[id]
	if !ok {
		return todo2.TodoDto{}, todo2.ErrNotFound
	}
	return t, nil
}

func (f *fakeRepository) List(_ todo2.ListTodosQuery) ([]todo2.TodoDto, error) {
	todos := make([]todo2.TodoDto, 0, len(f.todos))
	for _, t := range f.todos {
		todos = append(todos, t)
	}
	return todos, nil
}

func TestServiceWithInjectedRepository(t *testing.T) {
	validate, err := todo2.NewValidator()
	if err != nil {
		t.Fatalf("build validator: %v", err)
	}
	svc := todo2.NewService(newFakeRepository(), validate)

	created, err := svc.Create(todo2.CreateTodoRequest{Title: "Task"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.Title != "Task" || created.ID == 0 {
		t.Fatalf("unexpected created todo: %+v", created)
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("got ID %d, want %d", got.ID, created.ID)
	}
}

// TestCreateValidation exercises the declarative rules in NewValidator against
// the service boundary rather than through HTTP binding.
func TestCreateValidation(t *testing.T) {
	svc := newTestService(t)

	tests := []struct {
		name  string
		input todo2.CreateTodoRequest
	}{
		{"empty title", todo2.CreateTodoRequest{Title: ""}},
		{"blank title", todo2.CreateTodoRequest{Title: "   "}},
		{"title too long", todo2.CreateTodoRequest{Title: strings.Repeat("a", 201)}},
		{"invalid status", todo2.CreateTodoRequest{Title: "Task", Status: todo2.TodoStatus("nope")}},
		{"done without title", todo2.CreateTodoRequest{Title: " ", Status: todo2.Done}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Create(tc.input); !errors.Is(err, todo2.ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

// TestUpdateValidation covers the declarative rules applied to a patch payload.
func TestUpdateValidation(t *testing.T) {
	svc := newTestService(t)

	created, err := svc.Create(todo2.CreateTodoRequest{Title: "Task"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	blank := "   "
	tooLong := strings.Repeat("a", 201)
	badStatus := todo2.TodoStatus("nope")

	tests := []struct {
		name  string
		input todo2.UpdateTodoRequest
	}{
		{"blank title", todo2.UpdateTodoRequest{ID: created.ID, Title: &blank}},
		{"title too long", todo2.UpdateTodoRequest{ID: created.ID, Title: &tooLong}},
		{"invalid status", todo2.UpdateTodoRequest{ID: created.ID, Status: &badStatus}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Update(tc.input); !errors.Is(err, todo2.ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}
