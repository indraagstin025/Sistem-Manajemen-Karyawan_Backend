package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/router"
	_ "Sistem-Manajemen-Karyawan/docs" 
	_ "time/tzdata" 
)

// @title Sistem Manajemen Karyawan API
// @version 1.0
// @description API untuk sistem manajemen karyawan dengan fitur attendance, leave request, dan manajemen user
// @termsOfService https://github.com/your-repo/terms/
//
// @contact.name API Support
// @contact.url https://github.com/your-repo
// @contact.email support@example.com
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:3000
// @BasePath /api/v1
// @schemes http https
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
//
// @tag.name Auth
// @tag.description Authentication endpoints
//
// @tag.name Users
// @tag.description User management endpoints
//
// @tag.name Admin
// @tag.description Admin only endpoints
//
// @tag.name Departments
// @tag.description Department management endpoints
//
// @tag.name Attendance
// @tag.description Attendance management endpoints
//
// @tag.name Leave Request
// @tag.description Leave request management endpoints
func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file tidak ditemukan, menggunakan environment variables sistem")
	}

	cfg := config.LoadConfig() 

	config.MongoConnect()
	config.InitDatabase() 

	defer config.DisconnectDB()

	app := fiber.New()

	
	config.SetupCORS(app) 

	app.Use(logger.New())

	router.SetupRoutes(app) 

	log.Printf("Server running on port %s", cfg.Port)
	log.Printf("API Documentation: http://localhost:%s/docs/index.html", cfg.Port)
	log.Printf("Health Check: http://localhost:%s/", cfg.Port)
	log.Printf("CORS enabled for origins: %v", config.GetAllowedOrigins()) 
	log.Fatal(app.Listen(":" + cfg.Port))
}

