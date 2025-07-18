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

func SetupRoutes(
	app *fiber.App,
	userRepo *repository.UserRepository,                 // Ini adalah pointer ke struct, jadi (*) sudah benar
	deptRepo repository.DepartmentRepository,             // Ini adalah interface, JANGAN pakai (*)
	attendanceRepo repository.AttendanceRepository,         // Ini adalah interface, JANGAN pakai (*)
	leaveRepo repository.LeaveRequestRepository,            // Ini adalah interface, JANGAN pakai (*)
	workScheduleRepo *repository.WorkScheduleRepository, // Ini adalah pointer ke struct, jadi (*) sudah benar
) {
	log.Println("Memulai pendaftaran rute aplikasi...")

	// Inisialisasi Handlers (tidak ada perubahan di sini)
	authHandler := handlers.NewAuthHandler(userRepo)
	userHandler := handlers.NewUserHandler(userRepo, deptRepo, leaveRepo)
	deptHandler := handlers.NewDepartmentHandler(deptRepo)
	attendanceHandler := handlers.NewAttendanceHandler(attendanceRepo, workScheduleRepo)
	leaveHandler := handlers.NewLeaveRequestHandler(leaveRepo, attendanceRepo)
	fileHandler := handlers.NewFileHandler()
	workScheduleHandler := handlers.NewWorkScheduleHandler(workScheduleRepo)

	// Rute Health check & Dokumentasi
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Sistem Manajemen Karyawan API",
			"status":  "running",
			"docs":    "/docs/index.html",
		})
	})
	app.Get("/docs/*", swagger.HandlerDefault)
	app.Static("/uploads", "./uploads") // Melayani file statis dari folder uploads

	// Grup API v1
	api := app.Group("/api/v1")

	// Rute untuk mengakses file (membutuhkan login)
	api.Get("/files/:id", middleware.AuthMiddleware(), fileHandler.GetFileFromGridFS)
	api.Get("/attachments/:filename", middleware.AuthMiddleware(), fileHandler.GetFileByFilename)

	// Rute Autentikasi
	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/logout", middleware.AuthMiddleware(), authHandler.Logout)


	// Rute Pengguna (dilindungi otentikasi)
	protectedUserGroup := api.Group("/users", middleware.AuthMiddleware())
	protectedUserGroup.Post("/change-password", authHandler.ChangePassword)
	protectedUserGroup.Get("/:id", userHandler.GetUserByID)
	protectedUserGroup.Put("/:id", userHandler.UpdateUser)
	protectedUserGroup.Post("/:id/upload-photo", userHandler.UploadProfilePhoto)
	protectedUserGroup.Get("/:id/photo", userHandler.GetProfilePhoto)

	// Rute Admin (dilindungi otentikasi & middleware admin)
	adminGroup := api.Group("/admin", middleware.AuthMiddleware(), middleware.AdminMiddleware())
	adminGroup.Get("/users", userHandler.GetAllUsers)
	adminGroup.Delete("/users/:id", userHandler.DeleteUser)
	adminGroup.Get("/dashboard-stats", userHandler.GetDashboardStats)

	// Rute Departemen
	api.Get("/departments", middleware.AuthMiddleware(), deptHandler.GetAllDepartments) // Dapat diakses semua role terautentikasi
	api.Get("/departments/:id", middleware.AuthMiddleware(), deptHandler.GetDepartmentByID) // Dapat diakses semua role terautentikasi
	adminGroup.Post("/departments", deptHandler.CreateDepartment)
	adminGroup.Put("/departments/:id", deptHandler.UpdateDepartment)
	adminGroup.Delete("/departments/:id", deptHandler.DeleteDepartment)

	// Rute Kehadiran
	attendanceGroup := api.Group("/attendance", middleware.AuthMiddleware())
	attendanceGroup.Post("/scan", attendanceHandler.ScanQRCode)
	attendanceGroup.Get("/my-history", attendanceHandler.GetMyAttendanceHistory)
	attendanceGroup.Get("/my-today", attendanceHandler.GetMyTodayAttendance) // Endpoint untuk karyawan melihat status absensi hari ini

	adminAttendanceGroup := attendanceGroup.Group("/", middleware.AdminMiddleware()) // Grup khusus admin untuk absensi
	adminAttendanceGroup.Get("/generate-qr", attendanceHandler.GenerateQRCode)
	adminAttendanceGroup.Get("/today", attendanceHandler.GetTodayAttendance) // Laporan absensi hari ini untuk admin
	adminAttendanceGroup.Get("/history", attendanceHandler.GetAttendanceHistoryForAdmin) // Riwayat absensi semua karyawan untuk admin

	// Rute Pengajuan Cuti & Izin
	leaveGroup := api.Group("/leave-requests", middleware.AuthMiddleware())
	leaveGroup.Post("/", leaveHandler.CreateLeaveRequest)
	leaveGroup.Post("/:id/attachment", leaveHandler.UploadAttachment)
	leaveGroup.Get("/my-requests", leaveHandler.GetMyLeaveRequests)
	leaveGroup.Get("/summary", leaveHandler.GetLeaveSummary)

	adminLeaveGroup := leaveGroup.Group("/", middleware.AdminMiddleware()) // Grup khusus admin untuk cuti/izin
	adminLeaveGroup.Get("/", leaveHandler.GetAllLeaveRequests)
	adminLeaveGroup.Put("/:id/status", leaveHandler.UpdateLeaveRequestStatus)

	// ======================================================
	// Rute Jadwal Kerja (Work Schedules) - Diperbarui
	// ======================================================
	workScheduleGroup := api.Group("/work-schedules", middleware.AuthMiddleware())

	// Rute untuk SEMUA PENGGUNA TERAUTENTIKASI (Admin & Karyawan) untuk MELIHAT jadwal
	// Endpoint ini cerdas dan otomatis memfilter hari libur.
	workScheduleGroup.Get("/", workScheduleHandler.GetAllWorkSchedules)

	// Rute KHUSUS ADMIN untuk MENGELOLA (Create, Update, Delete) aturan jadwal
	workScheduleGroup.Post("/", middleware.AdminMiddleware(), workScheduleHandler.CreateWorkSchedule)
	workScheduleGroup.Put("/:id", middleware.AdminMiddleware(), workScheduleHandler.UpdateWorkSchedule)
	workScheduleGroup.Delete("/:id", middleware.AdminMiddleware(), workScheduleHandler.DeleteWorkSchedule)
	workScheduleGroup.Get("/:id", middleware.AdminMiddleware(), workScheduleHandler.GetWorkScheduleById) 
    
    // ======================================================
    // RUTE HARI LIBUR - INI YANG HILANG DAN PERLU DITAMBAHKAN!
    // ======================================================
    // Rute untuk mendapatkan daftar hari libur. Diproteksi dengan AuthMiddleware.
    api.Get("/holidays", middleware.AuthMiddleware(), workScheduleHandler.GetHolidays) // <--- TAMBAHKAN BARIS INI!


	log.Println("Semua rute aplikasi berhasil didaftarkan.")

	// ======================================================
	// LOG RUTE YANG TELAH DIPERBAIKI DAN DITAMBAHKAN
	// ======================================================
	log.Println("Routes yang tersedia:")
	log.Println("- GET /")
	log.Println("- GET /docs/*")
	log.Println("- GET /uploads (static files)")
	log.Println("- GET /api/v1/files/:id (protected)")
	log.Println("- GET /api/v1/attachments/:filename (protected)")

	log.Println("- POST /api/v1/auth/register")
	log.Println("- POST /api/v1/auth/login")

	log.Println("- POST /api/v1/users/change-password (protected)")
	log.Println("- GET /api/v1/users/:id (protected)")
	log.Println("- PUT /api/v1/users/:id (protected)")
	log.Println("- POST /api/v1/users/:id/upload-photo (protected)")
	log.Println("- GET /api/v1/users/:id/photo (protected)")

	log.Println("- GET /api/v1/admin/users (admin only)")
	log.Println("- DELETE /api/v1/admin/users/:id (admin only)")
	log.Println("- GET /api/v1/admin/dashboard-stats (admin only)")

	log.Println("- GET /api/v1/departments (protected)")
	log.Println("- GET /api/v1/departments/:id (protected)")
	log.Println("- POST /api/v1/admin/departments (admin only)")
	log.Println("- PUT /api/v1/admin/departments/:id (admin only)")
	log.Println("- DELETE /api/v1/admin/departments/:id (admin only)")

	log.Println("- POST /api/v1/attendance/scan (protected)")
	log.Println("- GET /api/v1/attendance/my-history (protected)")
	log.Println("- GET /api/v1/attendance/my-today (protected)")
	log.Println("- GET /api/v1/admin/attendance/generate-qr (admin only)")
	log.Println("- GET /api/v1/admin/attendance/today (admin only)")
	log.Println("- GET /api/v1/admin/attendance/history (admin only)") 

	log.Println("- POST /api/v1/leave-requests (protected)")
	log.Println("- POST /api/v1/leave-requests/:id/attachment (protected)")
	log.Println("- GET /api/v1/leave-requests/my-requests (protected)")
	log.Println("- GET /api/v1/admin/leave-requests (admin only)")
	log.Println("- PUT /api/v1/admin/leave-requests/:id/status (admin only)")

	log.Println("- GET /api/v1/work-schedules (protected)")           
	log.Println("- GET /api/v1/work-schedules/:id (admin only)")      
	log.Println("- POST /api/v1/work-schedules (admin only)")         
	log.Println("- PUT /api/v1/work-schedules/:id (admin only)")      
	log.Println("- DELETE /api/v1/work-schedules/:id (admin only)")   
    log.Println("- GET /api/v1/holidays (protected)")                 


	log.Println("Swagger documentation tersedia di: /docs/index.html")
}