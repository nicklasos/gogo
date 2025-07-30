package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"myapp/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// GetTestTimestamp returns a test timestamp
func GetTestTimestamp() pgtype.Timestamp {
	return pgtype.Timestamp{Time: time.Now(), Valid: true}
}

// CreateTestCity creates a test city and returns it
func CreateTestCity(t *testing.T, ctx context.Context, tx pgx.Tx) *db.City {
	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	// Generate unique city name using timestamp
	cityName := fmt.Sprintf("Test City %d", time.Now().UnixNano())

	city := &db.City{
		Name:      cityName,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Use the transaction directly for raw SQL queries
	row := tx.QueryRow(ctx,
		"INSERT INTO cities (name, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id, name, created_at, updated_at",
		city.Name, city.CreatedAt, city.UpdatedAt)

	err := row.Scan(&city.ID, &city.Name, &city.CreatedAt, &city.UpdatedAt)
	require.NoError(t, err, "Failed to create test city")

	return city
}

// CreateTestCityWithName creates a test city with a custom name
func CreateTestCityWithName(t *testing.T, ctx context.Context, tx pgx.Tx, name string) *db.City {
	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	city := &db.City{
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Use the transaction directly for raw SQL queries
	row := tx.QueryRow(ctx,
		"INSERT INTO cities (name, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id, name, created_at, updated_at",
		city.Name, city.CreatedAt, city.UpdatedAt)

	err := row.Scan(&city.ID, &city.Name, &city.CreatedAt, &city.UpdatedAt)
	require.NoError(t, err, "Failed to create test city")

	return city
}
