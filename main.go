package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/router"
)

func main() {

	cfg := config.LoadConfig()

	config.MongoConnect()
	config.InitDatabase()

	defer config.DisconnectDB()

	app := fiber.New()
	app.Use(cors.New())
	app.Use(logger.New())

	router.SetupRoutes(app)

	log.Fatal(app.Listen(":" + cfg.Port))
}
