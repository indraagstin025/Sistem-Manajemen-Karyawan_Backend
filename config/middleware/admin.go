package middleware

import (
	"Sistem-Manajemen-Karyawan/models" 
	"github.com/gofiber/fiber/v2"
)

func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {

		
		// 1. Ambil 'user' dan lakukan type assertion ke *models.Claims
		claims, ok := c.Locals("user").(*models.Claims)
		if !ok {
			// Error ini terjadi jika AuthMiddleware gagal atau ada kesalahan tipe data
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tidak terautentikasi atau data sesi rusak"})
		}

		// ================== PERUBAHAN SELESAI ==================

		// Logika pengecekan role tetap sama
		if claims.Role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak. Hak akses admin diperlukan"})
		}

		return c.Next()
	}
}