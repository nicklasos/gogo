package cities

import "app/internal"

func RegisterRoutes(app *internal.App) {
	// Create service with only the dependencies it needs
	service := NewCitiesService(app.Queries)

	// Create handler with only the service it needs
	handler := NewHandler(service)

	app.Api.GET("/cities", handler.ListCities)
}
