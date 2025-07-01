package middleware

import (
	"strings"

	"Sistem-Manajemen-Karyawan/pkg/paseto"
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

		claims, err := paseto.ValidateToken(tokenString)
		if err != nil {

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token", "details": err.Error()})
		}

		c.Locals("user", claims)

		return c.Next()
	}
}
