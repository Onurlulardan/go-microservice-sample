package main

import (
	"log"

	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/database"
)

func main() {
	log.Println("🌱 Starting database seeding...")

	// Load configuration
	config.LoadConfig()

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDatabase()

	// Run seeding
	if err := database.SeedDatabase(); err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}

	// Create super admin
	if err := database.CreateSuperAdmin("admin@forgecrud.com", "admin123", "Super", "Admin"); err != nil {
		log.Fatalf("Failed to create super admin: %v", err)
	}

	log.Println("✅ Database seeding completed successfully!")
}
