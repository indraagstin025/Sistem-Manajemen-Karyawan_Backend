package config

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "strings"
)

var allowedOrigins = []string{
    "http://localhost:5173",
    "http://localhost:4173",
    "http://127.0.0.1:5173",
    "https://sistem-manajemen-karyawan-frontend.vercel.app",
}

func GetAllowedOrigins() []string {
    return allowedOrigins
}

func SetupCORS(app *fiber.App) {
    app.Use(cors.New(cors.Config{
        AllowOrigins:     joinOrigins(allowedOrigins), 
        AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
        AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
        AllowCredentials: true,
        ExposeHeaders:    "Content-Length, Content-Type",
    }))
}


func joinOrigins(origins []string) string {
    return strings.Join(origins, ", ")
}
