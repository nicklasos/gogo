package cities

import "myapp/internal"

func RegisterRoutes(app *internal.App) {
	handler := NewHandler(app)

	app.Api.GET("/cities", handler.ListCities)
}
