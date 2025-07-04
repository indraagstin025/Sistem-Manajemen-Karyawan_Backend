package router

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"

	"Sistem-Manajemen-Karyawan/config/middleware"
	"Sistem-Manajemen-Karyawan/handlers"
	"Sistem-Manajemen-Karyawan/repository"
	_ "Sistem-Manajemen-Karyawan/docs" // Import docs untuk swagger
)

func SetupRoutes(app *fiber.App) {
	log.Println("Memulai pendaftaran rute aplikasi...")

	// Inisialisasi Repositories
	userRepo := repository.NewUserRepository()
	deptRepo := repository.NewDepartmentRepository() // PASTIKAN BARIS INI ADA

	// Inisialisasi Handlers
	authHandler := handlers.NewAuthHandler(userRepo)
	userHandler := handlers.NewUserHandler(userRepo)
	deptHandler := handlers.NewDepartmentHandler(deptRepo) // PASTIKAN BARIS INI ADA


	// Health check endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Sistem Manajemen Karyawan API",
			"status":  "running",
			"docs":    "/docs/index.html",
		})
	})

	// Swagger documentation endpoint
	app.Get("/docs/*", swagger.HandlerDefault)
	app.Static("/uploads", "./uploads")

	// API v1 group
	api := app.Group("/api/v1")

	// Authentication routes (public)
	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)

	// User routes (protected)
	protectedUserGroup := api.Group("/users", middleware.AuthMiddleware())
	protectedUserGroup.Post("/change-password", authHandler.ChangePassword)
	protectedUserGroup.Get("/:id", userHandler.GetUserByID)
	protectedUserGroup.Put("/:id", userHandler.UpdateUser)
	protectedUserGroup.Post("/:id/upload-photo", userHandler.UploadProfilePhoto)

	// Admin routes (admin only)
	adminGroup := api.Group("/admin", middleware.AuthMiddleware(), middleware.AdminMiddleware())
	adminGroup.Get("/users", userHandler.GetAllUsers)
	adminGroup.Delete("/users/:id", userHandler.DeleteUser)
	adminGroup.Get("/dashboard-stats", userHandler.GetDashboardStats) // Rute Dashboard Stats

	// Department routes
	// PASTIKAN KEDUA BARIS INI ADA DAN TIDAK DIKOMENTARI:
	api.Get("/departments", middleware.AuthMiddleware(), deptHandler.GetAllDepartments)
	api.Get("/departments/:id", middleware.AuthMiddleware(), deptHandler.GetDepartmentByID) // Opsional, tapi disarankan ada

	// Rute CRUD departemen (hanya admin)
	adminGroup.Post("/departments", deptHandler.CreateDepartment)
	adminGroup.Put("/departments/:id", deptHandler.UpdateDepartment)
	adminGroup.Delete("/departments/:id", deptHandler.DeleteDepartment)


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
	log.Println("- GET /api/v1/departments (protected)") // Ini yang dipanggil frontend
	log.Println("- GET /api/v1/departments/:id (protected)")
	log.Println("Swagger documentation tersedia di: /docs/index.html")
}
