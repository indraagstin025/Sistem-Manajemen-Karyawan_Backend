package main

import (
	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/repository" // BARU: import repository
	"Sistem-Manajemen-Karyawan/router"
	"context" // BARU: import context
	
	"log"

	_ "Sistem-Manajemen-Karyawan/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3" // BARU: import library cron
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
		log.Println("Warning: .env file tidak ditemukan...")
	}

	cfg := config.LoadConfig()
	config.MongoConnect()
	config.InitDatabase()
	defer config.DisconnectDB()

	// =======================================================
	// Inisialisasi Repositories (Dibuat Sekali)
	// =======================================================
	log.Println("Menginisialisasi semua repositories...")
	userRepo := repository.NewUserRepository()
	attendanceRepo := repository.NewAttendanceRepository()
	workScheduleRepo := repository.NewWorkScheduleRepository()
	leaveRequestRepo := repository.NewLeaveRequestRepository()
	deptRepo := repository.NewDepartmentRepository() // <-- BARU: Tambahkan inisialisasi ini

	// =======================================================
	// Penyiapan Cron Job
	// =======================================================
	c := cron.New()
	_, err = c.AddFunc("0 17-22 * * *", func() {
		err := attendanceRepo.MarkAbsentEmployeesAsAlpha(
			context.Background(),
			userRepo,
			workScheduleRepo,
			leaveRequestRepo,
		)
		if err != nil {
			log.Println("❌ Error saat menjalankan cron job Alpha:", err)
		}
	})
	if err != nil {
		log.Fatal("Gagal menambahkan cron job:", err)
	}
	c.Start()
	log.Println("✅ Scheduler untuk status Alpha otomatis telah dimulai.")

	// =======================================================
	// Setup Fiber App
	// =======================================================
	app := fiber.New()
	config.SetupCORS(app)
	app.Use(logger.New())

	// DIUBAH: Oper instance deptRepo juga ke SetupRoutes
	router.SetupRoutes(app, userRepo, deptRepo, attendanceRepo, leaveRequestRepo, workScheduleRepo)

	log.Printf("Server running on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}


