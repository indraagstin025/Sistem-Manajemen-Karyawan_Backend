package router

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"

	"Sistem-Manajemen-Karyawan/config/middleware"
	"Sistem-Manajemen-Karyawan/handlers"
	"Sistem-Manajemen-Karyawan/repository"
	_ "Sistem-Manajemen-Karyawan/docs"
)

func SetupRoutes(app *fiber.App) {
	log.Println("Memulai pendaftaran rute aplikasi...")

	// Inisialisasi Repositories
	userRepo := repository.NewUserRepository()
	deptRepo := repository.NewDepartmentRepository()
	attendanceRepo := repository.NewAttendanceRepository()
	leaveRepo := repository.NewLeaveRequestRepository()

	// Inisialisasi Handlers
	authHandler := handlers.NewAuthHandler(userRepo)
	userHandler := handlers.NewUserHandler(userRepo)
	deptHandler := handlers.NewDepartmentHandler(deptRepo)
	attendanceHandler := handlers.NewAttendanceHandler(attendanceRepo)
	leaveHandler := handlers.NewLeaveRequestHandler(leaveRepo, attendanceRepo)

	// Health check & Docs
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Sistem Manajemen Karyawan API",
			"status":  "running",
			"docs":    "/docs/index.html",
		})
	})
	app.Get("/docs/*", swagger.HandlerDefault)
	app.Static("/uploads", "./uploads")

	// API v1 group
	api := app.Group("/api/v1")

	// Authentication routes
	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)

	// User routes
	protectedUserGroup := api.Group("/users", middleware.AuthMiddleware())
	protectedUserGroup.Post("/change-password", authHandler.ChangePassword)
	protectedUserGroup.Get("/:id", userHandler.GetUserByID)
	protectedUserGroup.Put("/:id", userHandler.UpdateUser)
	protectedUserGroup.Post("/:id/upload-photo", userHandler.UploadProfilePhoto)

	// Admin routes
	adminGroup := api.Group("/admin", middleware.AuthMiddleware(), middleware.AdminMiddleware())
	adminGroup.Get("/users", userHandler.GetAllUsers)
	adminGroup.Delete("/users/:id", userHandler.DeleteUser)
	adminGroup.Get("/dashboard-stats", userHandler.GetDashboardStats)

	// Department routes
	api.Get("/departments", middleware.AuthMiddleware(), deptHandler.GetAllDepartments)
	api.Get("/departments/:id", middleware.AuthMiddleware(), deptHandler.GetDepartmentByID)
	adminGroup.Post("/departments", deptHandler.CreateDepartment)
	adminGroup.Put("/departments/:id", deptHandler.UpdateDepartment)
	adminGroup.Delete("/departments/:id", deptHandler.DeleteDepartment)

	// ======================================================
	// PERBAIKAN: Rute Kehadiran Karyawan
	// ======================================================
	// Middleware autentikasi diterapkan ke seluruh grup
	attendanceGroup := api.Group("/attendance", middleware.AuthMiddleware())

	// Rute untuk semua pengguna yang sudah login
	attendanceGroup.Post("/scan", attendanceHandler.ScanQRCode)
	attendanceGroup.Get("/my-history", attendanceHandler.GetMyAttendanceHistory)

	// Rute khusus admin (middleware admin ditambahkan di sini)
	adminAttendanceGroup := attendanceGroup.Group("/", middleware.AdminMiddleware())
	adminAttendanceGroup.Get("/generate-qr", attendanceHandler.GenerateQRCode)
	adminAttendanceGroup.Get("/today", attendanceHandler.GetTodayAttendance)
	// ======================================================

	// Rute untuk Pengajuan Izin, Cuti, dan Sakit
	leaveGroup := api.Group("/leave-requests", middleware.AuthMiddleware())
	leaveGroup.Post("/", leaveHandler.CreateLeaveRequest)
	leaveGroup.Post("/:id/attachment", leaveHandler.UploadAttachment)
	adminLeaveGroup := leaveGroup.Group("/", middleware.AdminMiddleware())
	adminLeaveGroup.Get("/", leaveHandler.GetAllLeaveRequests)
	adminLeaveGroup.Put("/:id/status", leaveHandler.UpdateLeaveRequestStatus)

	log.Println("Semua rute aplikasi berhasil didaftarkan.")
	log.Println("Routes yang tersedia:")
	log.Println("- POST /api/v1/auth/register")
	log.Println("- POST /api/v1/auth/login")
	log.Println("- POST /api/v1/users/change-password (protected)")
	log.Println("- GET /api/v1/users/:id (protected)")
	log.Println("- PUT /api/v1/users/:id (protected)")
	log.Println("- POST /api/v1/users/:id/upload-photo (protected)")
	log.Println("- GET /api/v1/admin/users (admin only)")
	log.Println("- DELETE /api/v1/admin/users/:id (admin only)")
	log.Println("- GET /api/v1/admin/dashboard-stats (admin only)")
	log.Println("- POST /api/v1/admin/departments (admin only)")
	log.Println("- PUT /api/v1/admin/departments/:id (admin only)")
	log.Println("- DELETE /api/v1/admin/departments/:id (admin only)")
	log.Println("- GET /api/v1/departments (protected)")
	log.Println("- GET /api/v1/departments/:id (protected)")
	log.Println("- POST /api/v1/attendance/scan (protected)")
	log.Println("- GET /api/v1/attendance/my-history (protected)") // Sekarang sudah benar
	log.Println("- POST /api/v1/attendance/generate-qr (admin only)")
	log.Println("- GET /api/v1/attendance/today (admin only)")
	log.Println("- POST /api/v1/leave-requests (protected)")
	log.Println("- POST /api/v1/leave-requests/:id/attachment (protected)")
	log.Println("- GET /api/v1/leave-requests (admin only)")
	log.Println("- PUT /api/v1/leave-requests/:id/status (admin only)")
	log.Println("Swagger documentation tersedia di: /docs/index.html")
}