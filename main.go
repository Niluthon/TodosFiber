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
	db := database.Connect()

	// Create/update tables for the models.
	if err := db.AutoMigrate(&todo.Todo{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Wire the layers: database -> repository -> service -> handler.
	repo := todo.NewTodoRepository(db)
	svc := todo.NewTodoService(repo)
	handler := todo.NewTodoHandler(svc)

	app := fiber.New(
		fiber.Config{
			StructValidator: &structValidator{validator: validator.New()},
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
