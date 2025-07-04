package main

import (
    "log"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/joho/godotenv"

    "Sistem-Manajemen-Karyawan/config"
    "Sistem-Manajemen-Karyawan/router"
    _ "Sistem-Manajemen-Karyawan/docs" // Import docs untuk swagger
)

// @title Sistem Manajemen Karyawan API
// @version 1.0
// @description API untuk sistem manajemen karyawan dengan authentication dan authorization
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:3000
// @BasePath /api/v1
// @schemes http 
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {

    // Load .env file
    err := godotenv.Load()
    if err != nil {
        log.Println("Warning: .env file tidak ditemukan, menggunakan environment variables sistem")
    }

    cfg := config.LoadConfig()

    config.MongoConnect()
    config.InitDatabase()

    defer config.DisconnectDB()

	app := fiber.New()

    // Setup CORS menggunakan konfigurasi dari cors.go
    config.SetupCORS(app)
    
    app.Use(logger.New())

    // Setup routes (termasuk Swagger di dalamnya)
    router.SetupRoutes(app)

    log.Printf("Server running on port %s", cfg.Port)
    log.Printf("API Documentation: http://localhost:%s/docs/index.html", cfg.Port)
    log.Printf("Health Check: http://localhost:%s/", cfg.Port)
    log.Printf("CORS enabled for origins: %v", config.GetAllowedOrigins())
    log.Fatal(app.Listen(":" + cfg.Port))
}