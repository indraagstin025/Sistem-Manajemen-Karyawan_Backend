package router

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"Sistem-Manajemen-Karyawan/config/middleware"
	"Sistem-Manajemen-Karyawan/handlers"
	"Sistem-Manajemen-Karyawan/repository"
)

func SetupRoutes(app *fiber.App) {
	log.Println("Memulai pendaftaran rute aplikasi...")

	userRepo := repository.NewUserRepository()

	authHandler := handlers.NewAuthHandler(userRepo)
	userHandler := handlers.NewUserHandler(userRepo)

	api := app.Group("/api")

	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)

	protectedUserGroup := api.Group("/users", middleware.AuthMiddleware())
	protectedUserGroup.Post("/change-password", authHandler.ChangePassword)
	protectedUserGroup.Get("/:id", userHandler.GetUserByID)
	protectedUserGroup.Put("/:id", userHandler.UpdateUser)

	adminGroup := api.Group("/admin", middleware.AuthMiddleware(), middleware.AdminMiddleware())
	adminGroup.Get("/users", userHandler.GetAllUsers)
	adminGroup.Delete("/users/:id", userHandler.DeleteUser)

	log.Println("Semua rute aplikasi berhasil didaftarkan dengan prefix '/api'.")
}
