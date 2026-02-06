package main

import (
	"log"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/sheenazien8/deck/internal/deck"
	"github.com/sheenazien8/deck/internal/deck/ui"
	"github.com/sheenazien8/deck/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewUserResources() *deck.Resource {
	return &deck.Resource{
		Name:  "users",
		Model: models.User{},
		Form: []deck.Field{
			deck.Text("name").
				Required().
				Label("Full Name"),
		},
		Table: []deck.Column{
			deck.Col("email"),
			deck.Col("created_at"),
		},
	}
}

var UserResource = NewUserResources()

func main() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		ctx := templ.InitializeContext(c.UserContext())

		b := templ.GetBuffer()
		defer templ.ReleaseBuffer(b)

		if err := ui.Label("id", "Hello, Deck UI!").Render(ctx, b); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		html := "<!doctype html><html><head><meta charset=\"utf-8\"><title>Deck UI</title></head><body>" + b.String() + "</body></html>"
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	})

	log.Println("Starting Fiber server at http://localhost:8080/")
	log.Fatal(app.Listen(":8080"))
}
