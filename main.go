package main

import (

	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/router"

)

func main() {

	// 1. Load Configuration from .env
	cfg := config.LoadConfig()

	// 2. Connect to MongoDB
	config.MongoConnect()
	
	// Ensure MongoDB connection is closed when the app exits
	defer config.DisconnectDB()

	// 3. Initialize Fiber App
	app := fiber.New()

	// 4. Register Global Middlewares
	app.Use(cors.New())
	app.Use(logger.New())

	// 5. Setup Routes
	router.SetupRoutes(app)

	// 6. Start the server
	log.Fatal(app.Listen(":" + cfg.Port))
}