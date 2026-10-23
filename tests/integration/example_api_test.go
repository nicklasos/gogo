package integration

import (
	"fmt"
	"net/http"
	"testing"

	"app/internal/errs"
	"app/internal/example"
	"app/internal/factory"
	"app/tests/testkit"

	"github.com/stretchr/testify/assert"
)

func TestExampleAPI_CreateExample(t *testing.T) {
	t.Run("creates an example for the logged-in user", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).POST("/api/v1/examples", example.CreateExampleRequest{Title: "Test Example", Description: "Test Description"}).
			AssertOK().
			AssertJSONPath("data.title", "Test Example").
			AssertJSONPath("data.description", "Test Description").
			AssertJSONPath("data.user_id", user.ID)

		app.AssertDatabaseHas("examples", testkit.Row{"user_id": user.ID, "title": "Test Example"})
	})

	t.Run("requires a login", func(t *testing.T) {
		app := testkit.New(t)

		app.POST("/api/v1/examples", example.CreateExampleRequest{Title: "Test Example"}).AssertUnauthorized()

		app.AssertDatabaseCount("examples", 0, testkit.Row{"title": "Test Example"})
	})

	t.Run("requires a title", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).POST("/api/v1/examples", `{"description": "Test Description"}`).AssertInvalid("title")
	})
}

func TestExampleAPI_GetExample(t *testing.T) {
	t.Run("returns the example", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		record := app.Create.Example(user.ID)

		app.As(user).GET("/api/v1/examples/%d", record.ID).
			AssertOK().
			AssertJSONPath("data.id", record.ID).
			AssertJSONPath("data.title", record.Title)
	})

	t.Run("returns 404 for an unknown example", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).GET("/api/v1/examples/99999").AssertNotFound()
	})
}

func TestExampleAPI_ListExamples(t *testing.T) {
	t.Run("returns a page of the caller's examples", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		app.Create.Examples(user.ID, 5, "Example")

		app.As(user).GET("/api/v1/examples?page=1&page_size=3").
			AssertOK().
			AssertJSONCount("data", 3).
			AssertJSONPath("pagination.total", 5).
			AssertJSONPath("pagination.current_page", 1).
			AssertJSONPath("pagination.per_page", 3)
	})

	t.Run("returns an empty list when there are none", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).GET("/api/v1/examples").
			AssertOK().
			AssertJSONCount("data", 0).
			AssertJSONPath("pagination.total", 0)
	})
}

func TestExampleAPI_UpdateExample(t *testing.T) {
	update := example.UpdateExampleRequest{Title: "Updated Title", Description: "Updated Description"}

	t.Run("updates the example", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		record := app.Create.Example(user.ID)

		app.As(user).PUT("/api/v1/examples/%d", update, record.ID).
			AssertOK().
			AssertJSONPath("data.title", "Updated Title").
			AssertJSONPath("data.description", "Updated Description")
	})

	t.Run("returns 404 for an unknown example", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).PUT("/api/v1/examples/99999", update).AssertNotFound()
	})
}

func TestExampleAPI_DeleteExample(t *testing.T) {
	t.Run("deletes the example", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		record := app.Create.Example(user.ID)

		app.As(user).DELETE("/api/v1/examples/%d", record.ID).AssertOK()

		app.AssertDatabaseMissing("examples", testkit.Row{"id": record.ID})
		app.As(user).GET("/api/v1/examples/%d", record.ID).AssertNotFound()
	})

	t.Run("returns 404 for an unknown example", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).DELETE("/api/v1/examples/99999").AssertNotFound()
	})
}

func TestExampleAPI_ListCacheIsInvalidatedForEveryPageSize(t *testing.T) {
	app := testkit.New(t)
	user := app.Create.User()
	titles := func(path string) []string {
		var page example.PaginatedExamplesResponse
		app.As(user).GET(path).AssertOK().JSON(&page)
		out := make([]string, len(page.Data))
		for i, item := range page.Data {
			out[i] = item.Title
		}
		return out
	}

	// Page sizes and pages well outside anything a fixed list of keys would cover
	paths := []string{"/api/v1/examples?page=1&page_size=7", "/api/v1/examples?page=1&page_size=33", "/api/v1/examples?page=9&page_size=1"}
	for _, path := range paths {
		assert.Empty(t, titles(path), "warm the cache: %s", path)
	}

	var created example.ExampleDataResponse
	app.As(user).POST("/api/v1/examples", `{"title": "Fresh", "description": ""}`).JSON(&created)
	assert.Equal(t, []string{"Fresh"}, titles(paths[0]))
	assert.Equal(t, []string{"Fresh"}, titles(paths[1]))

	app.As(user).PUT("/api/v1/examples/%d", `{"title": "Renamed", "description": ""}`, created.Data.ID).AssertOK()
	assert.Equal(t, []string{"Renamed"}, titles(paths[0]))

	app.As(user).DELETE("/api/v1/examples/%d", created.Data.ID).AssertOK()
	for _, path := range paths {
		assert.Empty(t, titles(path), path)
	}
}

func TestExampleAPI_Policy(t *testing.T) {
	app := testkit.New(t)
	owner := app.Create.User()
	stranger := app.Create.User()
	admin := app.Create.User(factory.Admin())
	record := app.Create.Example(owner.ID, factory.WithTitle("Private notes"))

	path := fmt.Sprintf("/api/v1/examples/%d", record.ID)
	body := `{"title": "Changed", "description": ""}`

	app.As(owner).GET(path).AssertOK()
	app.As(admin).GET(path).AssertOK()

	for _, attempt := range []*testkit.Response{
		app.As(stranger).GET(path),
		app.As(stranger).PUT(path, body),
		app.As(stranger).DELETE(path),
		app.As(admin).PUT(path, body),
	} {
		attempt.AssertError(http.StatusForbidden, errs.ErrKeyForbidden)
	}

	app.AssertDatabaseHas("examples", testkit.Row{"id": record.ID, "title": "Private notes"})

	var list example.PaginatedExamplesResponse
	app.As(admin).GET("/api/v1/examples").JSON(&list)
	assert.Empty(t, list.Data, "the list is always the caller's own examples")

	app.As(admin).DELETE(path).AssertOK()
	app.As(owner).GET(path).AssertError(http.StatusNotFound, errs.ErrKeyExampleNotFound)
}
