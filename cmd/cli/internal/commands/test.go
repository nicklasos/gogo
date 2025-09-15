package commands

import (
	"context"
	"flag"
	"fmt"

	"app/cmd/cli/internal"
	"app/internal/cities"
)

// RunTest runs various tests
func RunTest(app *internal.CLIApp, args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("Usage: go run cmd/cli test [options]")
		fmt.Println()
		fmt.Println("Run various tests")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run cmd/cli test")
		fmt.Println("  go run cmd/cli --test test    # Use test database")
	}

	fs.Parse(args)

	ctx := context.Background()

	app.Logger.Info(ctx, "Starting CLI test command")

	// Test database connection
	fmt.Println("🔗 Testing database connection...")
	if err := app.Database.Ping(ctx); err != nil {
		app.Logger.Error(ctx, "Database ping failed", err)
		fmt.Printf("❌ Database connection failed: %v\n", err)
		return
	}
	fmt.Println("✅ Database connection successful")

	// Test cities service if it exists
	fmt.Println("🏙️ Testing cities service...")
	citiesService := cities.NewCitiesService(app.Queries)
	
	cityList, err := citiesService.ListCities(ctx)
	if err != nil {
		app.Logger.Error(ctx, "Failed to list cities", err)
		fmt.Printf("❌ Cities service test failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Cities service working - found %d cities\n", len(cityList))

	// Test basic query
	fmt.Println("📊 Testing basic database query...")
	var dbVersion string
	err = app.Database.QueryRow(ctx, "SELECT version()").Scan(&dbVersion)
	if err != nil {
		app.Logger.Error(ctx, "Failed to get database version", err)
		fmt.Printf("❌ Database query failed: %v\n", err)
		return
	}
	fmt.Println("✅ Database query successful")

	app.Logger.Info(ctx, "CLI test command completed successfully")
	fmt.Println("🎉 All tests passed!")
}