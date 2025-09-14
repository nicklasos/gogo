package cities

import (
	"context"

	"app/internal/db"
	apperrors "app/internal/errors"
)

// CitiesService contains business logic for cities operations
type CitiesService struct {
	queries *db.Queries
}

// NewCitiesService creates a new cities service
func NewCitiesService(queries *db.Queries) *CitiesService {
	return &CitiesService{
		queries: queries,
	}
}

// ListCities retrieves all cities
func (cs *CitiesService) ListCities(ctx context.Context) ([]db.City, error) {
	result, err := cs.queries.ListCities(ctx)
	if err != nil {
		return nil, apperrors.WrapInternal("failed to list cities", err)
	}
	if result == nil {
		return []db.City{}, nil
	}
	return result, nil
}

// GetCityByID retrieves a city by ID
func (cs *CitiesService) GetCityByID(ctx context.Context, id int32) (*db.City, error) {
	city, err := cs.queries.GetCityByID(ctx, id)
	if err != nil {
		return nil, apperrors.WrapInternal("failed to get city", err)
	}
	return &city, nil
}
