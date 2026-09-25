package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

func main() {
	app := fiber.New()

	api := app.Group("/api")

	v1 := api.Group("/v1")

	v1.Route("/tasks", func(router fiber.Router) {
		router.Post("/", func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"success": true,
			})
		})

		router.Get("/", func(c fiber.Ctx) error {
			return c.SendString("Tasks list")
		})

		router.Get("/:id", func(c fiber.Ctx) error {
			id := c.Params("id")
			return c.SendString(id)
		})

		router.Patch("/:id", func(c fiber.Ctx) error {
			id := c.Params("id")
			return c.SendString(id)
		})

		router.Delete("/:id", func(c fiber.Ctx) error {
			id := c.Params("id")
			return c.SendString(id)
		})
	})

	log.Fatal(app.Listen(":8000"))
}
