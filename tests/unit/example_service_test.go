package unit

import (
	"testing"

	"app/internal/example"
	"app/tests/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleService_CreateExample(t *testing.T) {
	t.Run("should create example successfully", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := example.NewExampleService(app.Queries, nil)

		createdExample, err := service.CreateExample(app.Ctx, testkit.Actor(user), "Test Title", "Test Description")

		require.NoError(t, err)
		assert.NotNil(t, createdExample)
		assert.Equal(t, user.ID, createdExample.UserID)
		assert.Equal(t, "Test Title", createdExample.Title)
		assert.Equal(t, "Test Description", createdExample.Description.String)
		assert.True(t, createdExample.ID > 0)
	})

	t.Run("should create example with empty description", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := example.NewExampleService(app.Queries, nil)

		createdExample, err := service.CreateExample(app.Ctx, testkit.Actor(user), "Test Title", "")

		require.NoError(t, err)
		assert.NotNil(t, createdExample)
		assert.Equal(t, "Test Title", createdExample.Title)
		assert.False(t, createdExample.Description.Valid)
	})
}

func TestExampleService_GetExample(t *testing.T) {
	t.Run("should get example successfully", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		testExample := app.Create.Example(user.ID)
		service := example.NewExampleService(app.Queries, nil)

		result, err := service.GetExample(app.Ctx, testkit.Actor(user), testExample.ID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testExample.ID, result.ID)
		assert.Equal(t, testExample.Title, result.Title)
		assert.Equal(t, user.ID, result.UserID)
	})

	t.Run("should return error when example not found", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := example.NewExampleService(app.Queries, nil)

		result, err := service.GetExample(app.Ctx, testkit.Actor(user), 99999)

		assert.Error(t, err)
		assert.Equal(t, example.ErrExampleNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("should return error when example belongs to different user", func(t *testing.T) {
		app := testkit.New(t)
		user1 := app.Create.User()
		user2 := app.Create.User()
		testExample := app.Create.Example(user1.ID)
		service := example.NewExampleService(app.Queries, nil)

		result, err := service.GetExample(app.Ctx, testkit.Actor(user2), testExample.ID)

		assert.Equal(t, example.ErrExampleForbidden, err)
		assert.Nil(t, result)
	})
}

func TestExampleService_UpdateExample(t *testing.T) {
	t.Run("should update example successfully", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		testExample := app.Create.Example(user.ID)
		service := example.NewExampleService(app.Queries, nil)

		updatedExample, err := service.UpdateExample(app.Ctx, testkit.Actor(user), testExample.ID, "Updated Title", "Updated Description")

		require.NoError(t, err)
		assert.NotNil(t, updatedExample)
		assert.Equal(t, testExample.ID, updatedExample.ID)
		assert.Equal(t, "Updated Title", updatedExample.Title)
		assert.Equal(t, "Updated Description", updatedExample.Description.String)
	})

	t.Run("should return error when example not found", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := example.NewExampleService(app.Queries, nil)

		result, err := service.UpdateExample(app.Ctx, testkit.Actor(user), 99999, "Title", "Description")

		assert.Error(t, err)
		assert.Equal(t, example.ErrExampleNotFound, err)
		assert.Nil(t, result)
	})
}

func TestExampleService_DeleteExample(t *testing.T) {
	t.Run("should delete example successfully", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		testExample := app.Create.Example(user.ID)
		service := example.NewExampleService(app.Queries, nil)

		err := service.DeleteExample(app.Ctx, testkit.Actor(user), testExample.ID)

		require.NoError(t, err)

		// Verify example is deleted
		_, err = service.GetExample(app.Ctx, testkit.Actor(user), testExample.ID)
		assert.Error(t, err)
		assert.Equal(t, example.ErrExampleNotFound, err)
	})

	t.Run("should return error when example not found", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := example.NewExampleService(app.Queries, nil)

		err := service.DeleteExample(app.Ctx, testkit.Actor(user), 99999)

		assert.Error(t, err)
		assert.Equal(t, example.ErrExampleNotFound, err)
	})
}

func TestExampleService_ListExamplesPaginated(t *testing.T) {
	t.Run("should list paginated examples", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		for i := 0; i < 5; i++ {
			app.Create.Example(user.ID)
		}
		service := example.NewExampleService(app.Queries, nil)

		result, err := service.ListExamplesPaginated(app.Ctx, user.ID, 1, 3)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(5), result.Total)
		assert.Equal(t, 3, len(result.Data))
	})

	t.Run("should handle pagination correctly", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		for i := 0; i < 5; i++ {
			app.Create.Example(user.ID)
		}
		service := example.NewExampleService(app.Queries, nil)

		result, err := service.ListExamplesPaginated(app.Ctx, user.ID, 2, 3)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(5), result.Total)
		assert.Equal(t, 2, len(result.Data))
	})

}
