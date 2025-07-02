package middleware

import (
	"github.com/gofiber/fiber/v2"
	"Sistem-Manajemen-Karyawan/pkg/paseto" 
)

func AdminMiddleware() fiber.Handler{
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("user").(*paseto.Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "tidak terautentikasi"})
		}

		if claims.Role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "akses ditolak. hak akses admin diperlukan"})
		}

			return c.Next()
	}


}