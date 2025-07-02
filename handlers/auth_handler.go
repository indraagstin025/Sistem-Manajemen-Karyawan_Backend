package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"

	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/pkg/paseto"
	"Sistem-Manajemen-Karyawan/pkg/password"
	"Sistem-Manajemen-Karyawan/pkg/utils"
	"Sistem-Manajemen-Karyawan/repository"
)

type AuthHandler struct {
	useRepo *repository.UserRepository
}

func NewAuthHandler(userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		useRepo: userRepo,
	}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var payload models.UserRegisterPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body", "details": err.Error()})
	}

	if errors := util.ValidateStruct(payload); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": errors})
	}

	hashedPassword, err := password.HashPassword(payload.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal hash password"})
	}

	newUser := &models.User{
		Name:         payload.Name,
		Email:        payload.Email,
		Password:     hashedPassword,
		Role:         payload.Role,
		Position:     payload.Position,
		Department:   payload.Department,
		BaseSalary:   payload.BaseSalary,
		Address:      payload.Address,
		Photo:        payload.Photo,
		IsFirstLogin: true,
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	result, err := h.useRepo.CreateUser(ctx, newUser)
	if err != nil {

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mendaftarkan user: %v", err)})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User berhasil didaftarkan (oleh admin)",
		"user_id": result.InsertedID,
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var payload models.UserLoginPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body", "details": err.Error()})
	}

	if errors := util.ValidateStruct(payload); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": errors})
	}
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	user, err := h.useRepo.FindUserByEmail(ctx, payload.Email)
	if err != nil {
		if err.Error() == "user tidak ditemukan" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "kredensial tidak valid"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal menemukan user: %v", err)})
	}

	if !password.CheckPasswordHash(payload.Password, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "kredensial tidak valid"})
	}

	token, err := paseto.GenerateToken(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal membuat token"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":        "Login berhasil",
		"token":          token,
		"user_id":        user.ID,
		"role":           user.Role,
		"is_first_login": user.IsFirstLogin,
	})
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {

	claims, ok := c.Locals("user").(*paseto.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "tidak terautentikasi atau klaim token tidak valid"})
	}

	var payload models.ChangePasswordPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if errors := util.ValidateStruct(payload); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": errors})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	user, err := h.useRepo.FindUserByID(ctx, claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "user tidak ditemukan"})
	}

	if !password.CheckPasswordHash(payload.OldPassword, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "password lama tidak cocok"})
	}

	newHashedPassword, err := password.HashPassword(payload.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal hash password baru"})
	}

	updateData := bson.M{
		"password":     newHashedPassword,
		"isFirstLogin": false,
	}

	_, err = h.useRepo.UpdateUser(ctx, claims.UserID, updateData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal update password: %v", err)})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "password berhasil diubah."})
}
