package integration

import (
	"context"
	"net/http"
	"testing"

	"myapp/internal/cities"
	"myapp/internal/db"
	"myapp/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCitiesAPI_ListCities(t *testing.T) {
	t.Run("should return 200 with cities when cities exist", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Create test cities
			helpers.CreateTestCityWithName(t, ctx, tx, "São Paulo")
			helpers.CreateTestCityWithName(t, ctx, tx, "München")
			helpers.CreateTestCityWithName(t, ctx, tx, "New York-Brooklyn")

			// Test: Get cities
			resp := server.GET("/api/v1/cities")

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response cities.CitiesResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotNil(t, response.Data)
			assert.Len(t, response.Data, 3)

			// Cities should be ordered by name (alphabetical)
			
		})
	})

	t.Run("should return 200 with empty data when no cities exist", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server (no cities created)
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Test: Get cities when none exist
			resp := server.GET("/api/v1/cities")

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response cities.CitiesResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotNil(t, response.Data)
			assert.Len(t, response.Data, 0)
		})
	})

	t.Run("should return complete city data including timestamps", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Create test city
			city := helpers.CreateTestCityWithName(t, ctx, tx, "Test City")

			// Test: Get cities
			resp := server.GET("/api/v1/cities")

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response cities.CitiesResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotNil(t, response.Data)
			assert.Len(t, response.Data, 1)

			// Verify complete city data
			retrievedCity := response.Data[0]
			assert.Equal(t, city.ID, retrievedCity.ID)
			assert.Equal(t, city.Name, retrievedCity.Name)
			assert.True(t, retrievedCity.CreatedAt.Valid)
			assert.True(t, retrievedCity.UpdatedAt.Valid)

			// Verify timestamps match
			assert.Equal(t, city.CreatedAt.Time.Unix(), retrievedCity.CreatedAt.Time.Unix())
			assert.Equal(t, city.UpdatedAt.Time.Unix(), retrievedCity.UpdatedAt.Time.Unix())
		})
	})

	t.Run("should return cities ordered by name", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Create cities in non-alphabetical order
			cityZ := helpers.CreateTestCityWithName(t, ctx, tx, "Zulu City")
			cityA := helpers.CreateTestCityWithName(t, ctx, tx, "Alpha City")
			cityM := helpers.CreateTestCityWithName(t, ctx, tx, "Mike City")

			// Test: Get cities
			resp := server.GET("/api/v1/cities")

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response cities.CitiesResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotNil(t, response.Data)
			assert.Len(t, response.Data, 3)

			// Cities should be ordered alphabetically by name
			assert.Equal(t, cityA.ID, response.Data[0].ID)
			assert.Equal(t, "Alpha City", response.Data[0].Name)
			assert.Equal(t, cityM.ID, response.Data[1].ID)
			assert.Equal(t, "Mike City", response.Data[1].Name)
			assert.Equal(t, cityZ.ID, response.Data[2].ID)
			assert.Equal(t, "Zulu City", response.Data[2].Name)
		})
	})

	t.Run("should handle many cities", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Create many cities
			const numCities = 20
			createdCities := make([]*db.City, numCities)

			for i := 0; i < numCities; i++ {
				createdCities[i] = helpers.CreateTestCity(t, ctx, tx)
			}

			// Test: Get cities
			resp := server.GET("/api/v1/cities")

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response cities.CitiesResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotNil(t, response.Data)
			assert.Len(t, response.Data, numCities)

			// Verify all cities are present
			responseIDs := make(map[int32]bool)
			for _, city := range response.Data {
				responseIDs[city.ID] = true
			}

			for _, createdCity := range createdCities {
				assert.True(t, responseIDs[createdCity.ID], "All created cities should be in response")
			}

			// Verify ordering by name
			for i := 0; i < len(response.Data)-1; i++ {
				assert.True(t, response.Data[i].Name <= response.Data[i+1].Name,
					"Cities should be ordered by name: %s <= %s",
					response.Data[i].Name, response.Data[i+1].Name)
			}
		})
	})

	t.Run("should handle cities with special characters in names", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Create cities with special characters
			helpers.CreateTestCityWithName(t, ctx, tx, "São Paulo")
			helpers.CreateTestCityWithName(t, ctx, tx, "München")
			helpers.CreateTestCityWithName(t, ctx, tx, "New York-Brooklyn")

			// Test: Get cities
			resp := server.GET("/api/v1/cities")

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response cities.CitiesResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotNil(t, response.Data)
			assert.Len(t, response.Data, 3)

			// Verify special characters are preserved
			cityNames := make([]string, len(response.Data))
			for i, city := range response.Data {
				cityNames[i] = city.Name
			}

			assert.Contains(t, cityNames, "São Paulo")
			assert.Contains(t, cityNames, "München")
			assert.Contains(t, cityNames, "New York-Brooklyn")
		})
	})

	t.Run("should return consistent data structure", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Create test city
			helpers.CreateTestCityWithName(t, ctx, tx, "Test City")

			// Test: Get cities
			resp := server.GET("/api/v1/cities")

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Check response headers
			assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

			// Assert: Parse and validate response body structure
			var response cities.CitiesResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			// Verify the response structure matches the expected format
			assert.NotNil(t, response.Data)
			assert.IsType(t, []cities.City{}, response.Data)

			// Verify individual city structure
			retrievedCity := response.Data[0]
			assert.IsType(t, int32(0), retrievedCity.ID)
			assert.IsType(t, "", retrievedCity.Name)
			assert.NotZero(t, retrievedCity.ID)
			assert.NotEmpty(t, retrievedCity.Name)
		})
	})

	t.Run("should handle concurrent requests", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Create test cities
			cityA := helpers.CreateTestCityWithName(t, ctx, tx, "City A")
			cityB := helpers.CreateTestCityWithName(t, ctx, tx, "City B")

			// Test: Make multiple requests
			const numRequests = 5
			for i := 0; i < numRequests; i++ {
				resp := server.GET("/api/v1/cities")
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response cities.CitiesResponse
				err := resp.JSON(&response)
				require.NoError(t, err)

				assert.Len(t, response.Data, 2)
				// Cities are ordered by name, so "City A" should be first.
				assert.Equal(t, cityA.ID, response.Data[0].ID)
				assert.Equal(t, cityB.ID, response.Data[1].ID)
			}
		})
	})
}
