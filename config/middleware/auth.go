package middleware

import (
	"Sistem-Manajemen-Karyawan/pkg/paseto" // Pastikan path ini benar
	"strings"

	"github.com/gofiber/fiber/v2"
)

func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authorization header is required"})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authorization header format must be Bearer <token>"})
		}

		tokenString := parts[1]

		// ================== PERUBAHAN DIMULAI DI SINI ==================

		// 1. Buat instance PasetoMaker baru
		pasetoMaker, err := paseto.NewPasetoMaker()
		if err != nil {
			// Jika gagal (misal: .env tidak ada), ini adalah error server
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Server error: tidak bisa memproses token"})
		}

		// 2. Panggil method ValidateToken dari instance maker
		claims, err := pasetoMaker.ValidateToken(tokenString)
		if err != nil {
			// Jika token tidak valid (kadaluarsa, signature salah, dll.)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token tidak valid atau telah kadaluarsa",
			})
		}

		// ================== PERUBAHAN SELESAI ==================

		// Simpan claims ke context untuk digunakan oleh handler/middleware selanjutnya
		c.Locals("user", claims)

		return c.Next()
	}
}