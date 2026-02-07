package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/sheenazien8/deck/internal/deck"
	"github.com/sheenazien8/deck/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("deck.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	var count int64
	db.Model(&models.User{}).Count(&count)
	if count == 0 {
		dummyUsers := []models.User{
			{Name: "John Doe", Email: "john@example.com"},
			{Name: "Jane Smith", Email: "jane@example.com"},
			{Name: "Bob Johnson", Email: "bob@example.com"},
			{Name: "Alice Williams", Email: "alice@example.com"},
			{Name: "Charlie Brown", Email: "charlie@example.com"},
			{Name: "Diana Prince", Email: "diana@example.com"},
			{Name: "Eve Anderson", Email: "eve@example.com"},
			{Name: "Frank Miller", Email: "frank@example.com"},
			{Name: "Grace Lee", Email: "grace@example.com"},
			{Name: "Henry Davis", Email: "henry@example.com"},
			{Name: "Ivy Chen", Email: "ivy@example.com"},
			{Name: "Jack Wilson", Email: "jack@example.com"},
			{Name: "Kate Moore", Email: "kate@example.com"},
			{Name: "Leo Martin", Email: "leo@example.com"},
			{Name: "Mary Taylor", Email: "mary@example.com"},
			{Name: "Nick Anderson", Email: "nick@example.com"},
			{Name: "Olivia Thomas", Email: "olivia@example.com"},
			{Name: "Paul Jackson", Email: "paul@example.com"},
			{Name: "Quinn White", Email: "quinn@example.com"},
			{Name: "Rose Harris", Email: "rose@example.com"},
			{Name: "Sam Clark", Email: "sam@example.com"},
			{Name: "Tina Lewis", Email: "tina@example.com"},
			{Name: "Uma Robinson", Email: "uma@example.com"},
			{Name: "Victor Walker", Email: "victor@example.com"},
			{Name: "Wendy Hall", Email: "wendy@example.com"},
		}
		for _, user := range dummyUsers {
			db.Create(&user)
		}
		log.Println("Database seeded with 25 dummy users")
	}

	UserResource := deck.NewResource[models.User](db, "users").
		WithTable(
			deck.NewTable[*models.User]().
				Column(deck.TextColumn("Name", func(u *models.User) string {
					return u.Name
				})).
				Column(deck.TextColumn("Email", func(u *models.User) string {
					return u.Email
				})).
				SetActions(
					deck.EditAction[*models.User]("users"),
					deck.DeleteAction[*models.User]("users"),
				),
		).
		WithFields(
			deck.Text("name").Label("Name").Required(),
			deck.Email("email").Label("Email").Required(),
		)

	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		req := c.Request()
		var total int
		var largestName string
		var largestValLen int
		req.Header.VisitAll(func(k, v []byte) {
			total += len(k) + len(v)
			if len(v) > largestValLen {
				largestValLen = len(v)
				largestName = string(k)
			}
		})
		cookie := c.Get("Cookie")
		if cookie != "" {
			log.Printf("headers: total=%d largest=%s largestValLen=%d cookie_len=%d", total, largestName, largestValLen, len(cookie))
		} else {
			log.Printf("headers: total=%d largest=%s largestValLen=%d", total, largestName, largestValLen)
		}
		return c.Next()
	})

	app.Get("/__debug/headers", func(c *fiber.Ctx) error {
		req := c.Request()
		var total int
		var largestName string
		var largestValLen int
		headers := map[string]string{}
		req.Header.VisitAll(func(k, v []byte) {
			kstr := string(k)
			vstr := string(v)
			headers[kstr] = vstr
			total += len(k) + len(v)
			if len(v) > largestValLen {
				largestValLen = len(v)
				largestName = kstr
			}
		})

		body := fmt.Sprintf("total=%d largest=%s largestValLen=%d\n", total, largestName, largestValLen)
		for k, v := range headers {
			body += fmt.Sprintf("%s: %s\n", k, v)
		}
		c.Set("Content-Type", "text/plain; charset=utf-8")
		return c.SendString(body)
	})

	UserResource.RegisterRoutes(app)

	log.Println("Starting Fiber server at http://localhost:8080/")
	log.Println("Users CRUD routes:")
	log.Println("  - GET    /users       - List all users")
	log.Println("  - GET    /users/new   - Create new user form")
	log.Println("  - POST   /users       - Create new user")
	log.Println("  - GET    /users/:id/edit - Edit user form")
	log.Println("  - POST   /users/:id   - Update user")
	log.Println("  - POST   /users/:id/delete - Delete user")
	log.Fatal(app.Listen(":8080"))
}
