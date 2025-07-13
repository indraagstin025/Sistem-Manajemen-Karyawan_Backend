package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/router"
	// HAPUS: "Sistem-Manajemen-Karyawan/handlers" // Tidak perlu import handlers di sini
	_ "Sistem-Manajemen-Karyawan/docs" // Import docs untuk swagger
	_ "time/tzdata" // ✨ TAMBAHKAN BARIS INI! ✨
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

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file tidak ditemukan, menggunakan environment variables sistem")
	}

	cfg := config.LoadConfig() // Pastikan LoadConfig() ada dan berfungsi

	config.MongoConnect()
	config.InitDatabase() // Pastikan InitDatabase() dipanggil untuk membuat indeks, dll.

	defer config.DisconnectDB()

	app := fiber.New()

	// Setup CORS menggunakan konfigurasi dari cors.go
	config.SetupCORS(app) // Pastikan SetupCORS() ada dan berfungsi

	app.Use(logger.New())

	// Setup routes (termasuk Swagger di dalamnya)
	router.SetupRoutes(app) // Ini akan mendaftarkan semua rute Anda

	log.Printf("Server running on port %s", cfg.Port)
	log.Printf("API Documentation: http://localhost:%s/docs/index.html", cfg.Port)
	log.Printf("Health Check: http://localhost:%s/", cfg.Port)
	log.Printf("CORS enabled for origins: %v", config.GetAllowedOrigins()) // Pastikan GetAllowedOrigins() ada
	log.Fatal(app.Listen(":" + cfg.Port))
}

