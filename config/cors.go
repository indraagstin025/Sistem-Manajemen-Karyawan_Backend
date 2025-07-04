package config

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
)

var allowedOrigins = []string{
    "http://localhost:3000",        // Backend API
    "http://localhost:5173",        // Frontend Vite
    "http://localhost:8080",        // Frontend alternatif
    "http://127.0.0.1:3000",        // Backend API (localhost variant)
    "http://127.0.0.1:5173",        // Frontend alternatif
}

func GetAllowedOrigins() []string {
    return allowedOrigins
}

func SetupCORS(app *fiber.App) {
    app.Use(cors.New(cors.Config{
        AllowOrigins:     "*", // Temporary: allow all for testing
        AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
        AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
        AllowCredentials: false, // Set to false when using "*"
        ExposeHeaders:    "Content-Length, Content-Type",
    }))
}

func joinOrigins(origins []string) string {
    result := ""
    for i, origin := range origins {
        if i > 0 {
            result += ", "
        }
        result += origin
    }
    return result
}