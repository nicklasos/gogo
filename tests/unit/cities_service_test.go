package unit

import (
	"context"
	"testing"

	"app/internal/cities"
	"app/internal/db"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCitiesService_ListCities(t *testing.T) {
	t.Run("should return all cities ordered by name", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test cities
			city1 := helpers.CreateTestCity(t, ctx, tx)
			city2 := helpers.CreateTestCity(t, ctx, tx)
			city3 := helpers.CreateTestCity(t, ctx, tx)

			// Setup: Create cities service
			service := cities.NewCitiesService(queries)

			// Test: List all cities
			result, err := service.ListCities(ctx)

			// Assert: Verify result
			require.NoError(t, err)
			assert.Len(t, result, 3)

			// Cities should be ordered by name (ASC)
			// Note: CreateTestCity generates unique names with timestamps, so we can't predict exact order
			// but we can verify all cities are present
			cityIDs := []int32{result[0].ID, result[1].ID, result[2].ID}
			assert.Contains(t, cityIDs, city1.ID)
			assert.Contains(t, cityIDs, city2.ID)
			assert.Contains(t, cityIDs, city3.ID)

			// Verify ordering by name
			for i := 0; i < len(result)-1; i++ {
				assert.True(t, result[i].Name <= result[i+1].Name, "Cities should be ordered by name")
			}
		})
	})

	t.Run("should return empty slice when no cities exist", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create cities service (no cities created)
			service := cities.NewCitiesService(queries)

			// Test: List cities when none exist
			result, err := service.ListCities(ctx)

			// Assert: Should return empty slice, not error
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result, 0)
		})
	})

	t.Run("should return all cities with complete data", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test city
			city := helpers.CreateTestCity(t, ctx, tx)

			// Setup: Create cities service
			service := cities.NewCitiesService(queries)

			// Test: List cities
			result, err := service.ListCities(ctx)

			// Assert: Verify result
			require.NoError(t, err)
			assert.Len(t, result, 1)

			// Verify complete city data
			retrievedCity := result[0]
			assert.Equal(t, city.ID, retrievedCity.ID)
			assert.Equal(t, city.Name, retrievedCity.Name)
			assert.True(t, retrievedCity.CreatedAt.Valid)
			assert.True(t, retrievedCity.UpdatedAt.Valid)
			assert.Equal(t, city.CreatedAt.Time, retrievedCity.CreatedAt.Time)
			assert.Equal(t, city.UpdatedAt.Time, retrievedCity.UpdatedAt.Time)
		})
	})

	t.Run("should handle multiple cities with different names", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test cities with specific names for predictable ordering
			cityA := helpers.CreateTestCityWithName(t, ctx, tx, "Alpha City")
			cityB := helpers.CreateTestCityWithName(t, ctx, tx, "Beta City")
			cityC := helpers.CreateTestCityWithName(t, ctx, tx, "Gamma City")

			// Setup: Create cities service
			service := cities.NewCitiesService(queries)

			// Test: List cities
			result, err := service.ListCities(ctx)

			// Assert: Verify result
			require.NoError(t, err)
			assert.Len(t, result, 3)

			// Verify ordering by name (alphabetical)
			assert.Equal(t, cityA.ID, result[0].ID)
			assert.Equal(t, "Alpha City", result[0].Name)
			assert.Equal(t, cityB.ID, result[1].ID)
			assert.Equal(t, "Beta City", result[1].Name)
			assert.Equal(t, cityC.ID, result[2].ID)
			assert.Equal(t, "Gamma City", result[2].Name)
		})
	})

	t.Run("should handle large number of cities", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create many cities
			const numCities = 10
			createdCities := make([]*db.City, numCities)

			for i := 0; i < numCities; i++ {
				createdCities[i] = helpers.CreateTestCity(t, ctx, tx)
			}

			// Setup: Create cities service
			service := cities.NewCitiesService(queries)

			// Test: List cities
			result, err := service.ListCities(ctx)

			// Assert: Verify result
			require.NoError(t, err)
			assert.Len(t, result, numCities)

			// Verify all cities are present
			resultIDs := make(map[int32]bool)
			for _, city := range result {
				resultIDs[city.ID] = true
			}

			for _, createdCity := range createdCities {
				assert.True(t, resultIDs[createdCity.ID], "All created cities should be in result")
			}

			// Verify ordering by name
			for i := 0; i < len(result)-1; i++ {
				assert.True(t, result[i].Name <= result[i+1].Name, "Cities should be ordered by name")
			}
		})
	})

	t.Run("should handle cities with same name prefix", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create cities with similar names
			city1 := helpers.CreateTestCityWithName(t, ctx, tx, "New York")
			city2 := helpers.CreateTestCityWithName(t, ctx, tx, "New Orleans")
			city3 := helpers.CreateTestCityWithName(t, ctx, tx, "New Jersey")

			// Setup: Create cities service
			service := cities.NewCitiesService(queries)

			// Test: List cities
			result, err := service.ListCities(ctx)

			// Assert: Verify result
			require.NoError(t, err)
			assert.Len(t, result, 3)

			// Verify ordering by name (alphabetical)
			assert.Equal(t, city3.ID, result[0].ID) // New Jersey
			assert.Equal(t, "New Jersey", result[0].Name)
			assert.Equal(t, city2.ID, result[1].ID) // New Orleans
			assert.Equal(t, "New Orleans", result[1].Name)
			assert.Equal(t, city1.ID, result[2].ID) // New York
			assert.Equal(t, "New York", result[2].Name)
		})
	})
}
