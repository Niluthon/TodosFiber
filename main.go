package main

import (
	"log"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"

	"TodosFiber/database"
	"TodosFiber/features/todo"
)

type structValidator struct {
	validator *validator.Validate
}

func (v *structValidator) Validate(out interface{}) error { return v.validator.Struct(out) }

func main() {

	if err := os.Setenv("PORT", "8000"); err != nil {
		return
	}
	if err := os.Setenv("DB_DSN", "todos.db"); err != nil {
		return
	}

	db := database.Connect()

	// Create/update tables for the models.
	if err := db.AutoMigrate(&todo.TodoGorm{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Build one validator shared by Fiber request binding and the service's
	// business-rule validation so both layers agree on the rules.
	validate, err := todo.NewValidator()
	if err != nil {
		log.Fatalf("failed to build validator: %v", err)
	}

	// Wire the layers: database -> repository -> service -> handler.
	repo := todo.NewTodoRepository(db)
	svc := todo.NewService(repo, validate)
	handler := todo.NewTodoHandler(svc)

	app := fiber.New(
		fiber.Config{
			StructValidator: &structValidator{validator: validate},
		},
	)
	app.Use(logger.New())

	tasks := app.Group("/tasks")
	tasks.Post("/", handler.Create)
	tasks.Get("/", handler.List)
	tasks.Get("/:id", handler.Get)
	tasks.Patch("/:id", handler.Update)
	tasks.Delete("/:id", handler.Delete)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Fatal(app.Listen(":" + port))
}
