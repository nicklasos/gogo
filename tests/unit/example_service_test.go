package unit

import (
	"context"
	"testing"

	"app/internal/db"
	"app/internal/example"
	"app/internal/factory"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleService_CreateExample(t *testing.T) {
	t.Run("should create example successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := factory.User(t, tx)
			service := example.NewExampleService(queries, nil)

			// Test: Create example
			createdExample, err := service.CreateExample(ctx, helpers.Actor(user), "Test Title", "Test Description")

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, createdExample)
			assert.Equal(t, user.ID, createdExample.UserID)
			assert.Equal(t, "Test Title", createdExample.Title)
			assert.Equal(t, "Test Description", createdExample.Description.String)
			assert.True(t, createdExample.ID > 0)
		})
	})

	t.Run("should create example with empty description", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := factory.User(t, tx)
			service := example.NewExampleService(queries, nil)

			// Test: Create example with empty description
			createdExample, err := service.CreateExample(ctx, helpers.Actor(user), "Test Title", "")

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, createdExample)
			assert.Equal(t, "Test Title", createdExample.Title)
			assert.False(t, createdExample.Description.Valid)
		})
	})
}

func TestExampleService_GetExample(t *testing.T) {
	t.Run("should get example successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and example
			user := factory.User(t, tx)
			testExample := factory.Example(t, tx, user.ID)
			service := example.NewExampleService(queries, nil)

			// Test: Get example
			result, err := service.GetExample(ctx, helpers.Actor(user), testExample.ID)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, testExample.ID, result.ID)
			assert.Equal(t, testExample.Title, result.Title)
			assert.Equal(t, user.ID, result.UserID)
		})
	})

	t.Run("should return error when example not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := factory.User(t, tx)
			service := example.NewExampleService(queries, nil)

			// Test: Get non-existent example
			result, err := service.GetExample(ctx, helpers.Actor(user), 99999)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, example.ErrExampleNotFound, err)
			assert.Nil(t, result)
		})
	})

	t.Run("should return error when example belongs to different user", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create two users and example for first user
			user1 := factory.User(t, tx)
			user2 := factory.User(t, tx)
			testExample := factory.Example(t, tx, user1.ID)
			service := example.NewExampleService(queries, nil)

			// Test: Try to get example with different user ID
			result, err := service.GetExample(ctx, helpers.Actor(user2), testExample.ID)

			// Assert: it exists, but it is not theirs
			assert.Equal(t, example.ErrExampleForbidden, err)
			assert.Nil(t, result)
		})
	})
}

func TestExampleService_UpdateExample(t *testing.T) {
	t.Run("should update example successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and example
			user := factory.User(t, tx)
			testExample := factory.Example(t, tx, user.ID)
			service := example.NewExampleService(queries, nil)

			// Test: Update example
			updatedExample, err := service.UpdateExample(ctx, helpers.Actor(user), testExample.ID, "Updated Title", "Updated Description")

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, updatedExample)
			assert.Equal(t, testExample.ID, updatedExample.ID)
			assert.Equal(t, "Updated Title", updatedExample.Title)
			assert.Equal(t, "Updated Description", updatedExample.Description.String)
		})
	})

	t.Run("should return error when example not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := factory.User(t, tx)
			service := example.NewExampleService(queries, nil)

			// Test: Update non-existent example
			result, err := service.UpdateExample(ctx, helpers.Actor(user), 99999, "Title", "Description")

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, example.ErrExampleNotFound, err)
			assert.Nil(t, result)
		})
	})
}

func TestExampleService_DeleteExample(t *testing.T) {
	t.Run("should delete example successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and example
			user := factory.User(t, tx)
			testExample := factory.Example(t, tx, user.ID)
			service := example.NewExampleService(queries, nil)

			// Test: Delete example
			err := service.DeleteExample(ctx, helpers.Actor(user), testExample.ID)

			// Assert: Verify result
			require.NoError(t, err)

			// Verify example is deleted
			_, err = service.GetExample(ctx, helpers.Actor(user), testExample.ID)
			assert.Error(t, err)
			assert.Equal(t, example.ErrExampleNotFound, err)
		})
	})

	t.Run("should return error when example not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := factory.User(t, tx)
			service := example.NewExampleService(queries, nil)

			// Test: Delete non-existent example
			err := service.DeleteExample(ctx, helpers.Actor(user), 99999)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, example.ErrExampleNotFound, err)
		})
	})
}

func TestExampleService_ListExamplesPaginated(t *testing.T) {
	t.Run("should list paginated examples", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and multiple examples
			user := factory.User(t, tx)
			for i := 0; i < 5; i++ {
				factory.Example(t, tx, user.ID)
			}
			service := example.NewExampleService(queries, nil)

			// Test: List paginated examples
			result, err := service.ListExamplesPaginated(ctx, user.ID, 1, 3)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, int64(5), result.Total)
			assert.Equal(t, 3, len(result.Data))
		})
	})

	t.Run("should handle pagination correctly", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and multiple examples
			user := factory.User(t, tx)
			for i := 0; i < 5; i++ {
				factory.Example(t, tx, user.ID)
			}
			service := example.NewExampleService(queries, nil)

			// Test: Get second page
			result, err := service.ListExamplesPaginated(ctx, user.ID, 2, 3)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, int64(5), result.Total)
			assert.Equal(t, 2, len(result.Data))
		})
	})

}
