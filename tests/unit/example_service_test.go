package unit

import (
	"context"
	"testing"

	"app/internal/db"
	"app/internal/example"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleService_CreateExample(t *testing.T) {
	t.Run("should create example successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := helpers.CreateTestUser(t, ctx, tx)
			service := example.NewExampleService(queries)

			// Test: Create example
			createdExample, err := service.CreateExample(ctx, user.ID, "Test Title", "Test Description")

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
			user := helpers.CreateTestUser(t, ctx, tx)
			service := example.NewExampleService(queries)

			// Test: Create example with empty description
			createdExample, err := service.CreateExample(ctx, user.ID, "Test Title", "")

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
			user := helpers.CreateTestUser(t, ctx, tx)
			testExample := helpers.CreateTestExample(t, ctx, tx, user.ID)
			service := example.NewExampleService(queries)

			// Test: Get example
			result, err := service.GetExample(ctx, testExample.ID, user.ID)

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
			user := helpers.CreateTestUser(t, ctx, tx)
			service := example.NewExampleService(queries)

			// Test: Get non-existent example
			result, err := service.GetExample(ctx, 99999, user.ID)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, example.ErrExampleNotFound, err)
			assert.Nil(t, result)
		})
	})

	t.Run("should return error when example belongs to different user", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create two users and example for first user
			user1 := helpers.CreateTestUser(t, ctx, tx)
			user2 := helpers.CreateTestUser(t, ctx, tx)
			testExample := helpers.CreateTestExample(t, ctx, tx, user1.ID)
			service := example.NewExampleService(queries)

			// Test: Try to get example with different user ID
			result, err := service.GetExample(ctx, testExample.ID, user2.ID)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, example.ErrExampleNotFound, err)
			assert.Nil(t, result)
		})
	})
}

func TestExampleService_UpdateExample(t *testing.T) {
	t.Run("should update example successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and example
			user := helpers.CreateTestUser(t, ctx, tx)
			testExample := helpers.CreateTestExample(t, ctx, tx, user.ID)
			service := example.NewExampleService(queries)

			// Test: Update example
			updatedExample, err := service.UpdateExample(ctx, testExample.ID, user.ID, "Updated Title", "Updated Description")

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
			user := helpers.CreateTestUser(t, ctx, tx)
			service := example.NewExampleService(queries)

			// Test: Update non-existent example
			result, err := service.UpdateExample(ctx, 99999, user.ID, "Title", "Description")

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
			user := helpers.CreateTestUser(t, ctx, tx)
			testExample := helpers.CreateTestExample(t, ctx, tx, user.ID)
			service := example.NewExampleService(queries)

			// Test: Delete example
			err := service.DeleteExample(ctx, testExample.ID, user.ID)

			// Assert: Verify result
			require.NoError(t, err)

			// Verify example is deleted
			_, err = service.GetExample(ctx, testExample.ID, user.ID)
			assert.Error(t, err)
			assert.Equal(t, example.ErrExampleNotFound, err)
		})
	})

	t.Run("should return error when example not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := helpers.CreateTestUser(t, ctx, tx)
			service := example.NewExampleService(queries)

			// Test: Delete non-existent example
			err := service.DeleteExample(ctx, 99999, user.ID)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, example.ErrExampleNotFound, err)
		})
	})
}

func TestExampleService_ListExamples(t *testing.T) {
	t.Run("should list examples for user", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and multiple examples
			user := helpers.CreateTestUser(t, ctx, tx)
			example1 := helpers.CreateTestExample(t, ctx, tx, user.ID)
			example2 := helpers.CreateTestExample(t, ctx, tx, user.ID)
			service := example.NewExampleService(queries)

			// Test: List examples
			examples, err := service.ListExamples(ctx, user.ID)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, examples)
			assert.GreaterOrEqual(t, len(examples), 2)
			
			// Verify examples are in the list
			exampleIDs := make(map[int32]bool)
			for _, ex := range examples {
				exampleIDs[ex.ID] = true
			}
			assert.True(t, exampleIDs[example1.ID])
			assert.True(t, exampleIDs[example2.ID])
		})
	})

	t.Run("should return empty list when user has no examples", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := helpers.CreateTestUser(t, ctx, tx)
			service := example.NewExampleService(queries)

			// Test: List examples
			examples, err := service.ListExamples(ctx, user.ID)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, examples)
			assert.Equal(t, 0, len(examples))
		})
	})
}

func TestExampleService_ListExamplesPaginated(t *testing.T) {
	t.Run("should list paginated examples", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and multiple examples
			user := helpers.CreateTestUser(t, ctx, tx)
			for i := 0; i < 5; i++ {
				helpers.CreateTestExample(t, ctx, tx, user.ID)
			}
			service := example.NewExampleService(queries)

			// Test: List paginated examples
			result, err := service.ListExamplesPaginated(ctx, user.ID, 1, 3)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, int32(1), result.Page)
			assert.Equal(t, int32(3), result.PageSize)
			assert.Equal(t, int64(5), result.Total)
			assert.Equal(t, 3, len(result.Data))
		})
	})

	t.Run("should handle pagination correctly", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and multiple examples
			user := helpers.CreateTestUser(t, ctx, tx)
			for i := 0; i < 5; i++ {
				helpers.CreateTestExample(t, ctx, tx, user.ID)
			}
			service := example.NewExampleService(queries)

			// Test: Get second page
			result, err := service.ListExamplesPaginated(ctx, user.ID, 2, 3)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, int32(2), result.Page)
			assert.Equal(t, int32(3), result.PageSize)
			assert.Equal(t, int64(5), result.Total)
			assert.Equal(t, 2, len(result.Data))
		})
	})

	t.Run("should return error for invalid page", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := helpers.CreateTestUser(t, ctx, tx)
			service := example.NewExampleService(queries)

			// Test: List with invalid page
			result, err := service.ListExamplesPaginated(ctx, user.ID, 0, 10)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	})

	t.Run("should return error for invalid page size", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := helpers.CreateTestUser(t, ctx, tx)
			service := example.NewExampleService(queries)

			// Test: List with invalid page size
			result, err := service.ListExamplesPaginated(ctx, user.ID, 1, 101)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	})
}
