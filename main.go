// file: main.go

package main

import (
	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/repository"
	"Sistem-Manajemen-Karyawan/router"
	// "Sistem-Manajemen-Karyawan/seeder"
	// "Sistem-Manajemen-Karyawan/seeder"
	"context"

	"log"

	_ "Sistem-Manajemen-Karyawan/docs" // Import ini mungkin digunakan untuk Swagger

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	_ "time/tzdata" // Import ini mungkin digunakan untuk zona waktu
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
// @host https://sistem-manajemen-karyawanbackend-production.up.railway.app/
// @BasePath /api/v1
// @schemes https
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

	// =======================================================
	// Inisialisasi Repositories (Dibuat Sekali)
	// =======================================================
	log.Println("Menginisialisasi semua repositories...")
	userRepo := repository.NewUserRepository()
	attendanceRepo := repository.NewAttendanceRepository()
	workScheduleRepo := repository.NewWorkScheduleRepository()
	leaveRequestRepo := repository.NewLeaveRequestRepository()
	// Menggunakan 'deptRepo' sebagai nama variabel untuk DepartmentRepository
	deptRepo := repository.NewDepartmentRepository() 

	// =======================================================
	// Panggil Seeders (URUTAN PENTING DI SINI!)
	// =======================================================
	// log.Println("Memulai proses seeding data dummy...")
	// // Panggil Department Seeder DULU, agar data departemen tersedia
	// seeder.SeedDepartments(deptRepo)
	// // Kemudian panggil User Seeder, yang bergantung pada data departemen
	// seeder.SeedUsers(userRepo, deptRepo)
	// log.Println("Proses seeding data dummy selesai.")


	// =======================================================
	// Penyiapan Cron Job
	// =======================================================
	c := cron.New()
	// Pastikan scheduler dimulai setelah seeder, karena seeder butuh koneksi database
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

	// DIUBAH: Pastikan semua instance repository dioper ke SetupRoutes
	// Urutan argumen di sini harus sesuai dengan yang didefinisikan di router.SetupRoutes
	router.SetupRoutes(app, userRepo, deptRepo, attendanceRepo, leaveRequestRepo, workScheduleRepo) 

	log.Printf("Server running on port %s", cfg.Port)
	// log.Printf("API Documentation: http://localhost:%s/docs/index.html", cfg.Port) // Baris ini bisa diaktifkan jika perlu
	// log.Printf("Health Check: http://localhost:%s/", cfg.Port) // Baris ini bisa diaktifkan jika perlu
	// log.Printf("CORS enabled for origins: %v", config.GetAllowedOrigins()) // Baris ini bisa diaktifkan jika perlu
	log.Fatal(app.Listen(":" + cfg.Port))
}