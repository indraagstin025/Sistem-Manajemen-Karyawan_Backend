// Sistem-Manajemen-Karyawan/router/router.go
package router

import (
    "github.com/gofiber/fiber/v2"
    "log" // Tambahkan log untuk contoh sementara
)

// SetupRoutes mendaftarkan semua rute aplikasi Fiber.
func SetupRoutes(app *fiber.App) {
    log.Println("Setting up application routes...")
    // Contoh rute dasar (nanti akan Anda isi dengan rute sebenarnya)
    app.Get("/", func(c *fiber.Ctx) error {
        return c.SendString("Aplikasi Fiber berjalan!")
    })
}

