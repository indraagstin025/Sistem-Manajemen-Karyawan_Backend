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
	userRepo *repository.UserRepository
}

func NewAuthHandler(userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
	}
}

// Register godoc
// @Summary Register User
// @Description Mendaftarkan user baru (hanya admin yang dapat melakukan registrasi)
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body models.UserRegisterPayload true "Data registrasi user"
// @Success 201 {object} models.RegisterSuccessResponse "User berhasil didaftarkan"
// @Failure 400 {object} models.ValidationErrorResponse "Invalid request body atau validation error"
// @Failure 500 {object} models.ErrorResponse "Gagal hash password atau gagal mendaftarkan user"
// @Router /auth/register [post]
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

	result, err := h.userRepo.CreateUser(ctx, newUser)
	if err != nil {

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mendaftarkan user: %v", err)})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User berhasil didaftarkan (oleh admin)",
		"user_id": result.InsertedID,
	})
}

// Login
// @Summary Login User
// @Description Melakukan proses login dan mengembalikan token PASETO jika email dan password valid.
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body models.UserLoginPayload true "Kredensial untuk Login"
// @Success 200 {object} object{message=string, token=string, user=models.User} "Login Berhasil"
// @Failure 400 {object} object{error=string} "Payload tidak valid"
// @Failure 401 {object} object{error=string} "Kombinasi email dan password salah"
// @Failure 500 {object} object{error=string} "Error Internal Server"
// @Router /auth/login [post]
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

	user, err := h.userRepo.FindUserByEmail(ctx, payload.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Kombinasi email dan password salah"})
	}

	if !password.CheckPasswordHash(payload.Password, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Kombinasi email dan password salah"})
	}

	pasetoMaker, err := paseto.NewPasetoMaker()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menginisialisasi token generator"})
	}

	token, err := pasetoMaker.GenerateToken(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat token"})
	}


	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login berhasil",
		"token":   token,
		"user":    user,
	})
}

// ChangePassword godoc
// @Summary Change Password
// @Description Mengubah password user yang sedang login (required authentication)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param password body models.ChangePasswordPayload true "Data untuk mengubah password"
// @Success 200 {object} models.ChangePasswordSuccessResponse "Password berhasil diubah"
// @Failure 400 {object} models.ErrorResponse "Invalid request body"
// @Failure 401 {object} models.UnauthorizedErrorResponse "Tidak terautentikasi atau password lama tidak cocok"
// @Failure 500 {object} models.ErrorResponse "User tidak ditemukan atau gagal update"
// @Router /users/change-password [post]
// File: handlers/auth_handler.go

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tidak terautentikasi atau data sesi rusak"})
	}

	var payload models.ChangePasswordPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if errors := util.ValidateStruct(payload); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": errors})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	user, err := h.userRepo.FindUserByID(ctx, claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User tidak ditemukan"})
	}

	// 1. Cek dulu apakah password lama yang dimasukkan benar
	if !password.CheckPasswordHash(payload.OldPassword, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Password lama tidak cocok"})
	}

	if payload.NewPassword == payload.OldPassword {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password baru tidak boleh sama dengan password lama."})
	}

	newHashedPassword, err := password.HashPassword(payload.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal hash password baru"})
	}

	updateData := bson.M{
		"password":     newHashedPassword,
		"isFirstLogin": false,
		"updated_at":   time.Now(),
	}

	_, err = h.userRepo.UpdateUser(ctx, claims.UserID, updateData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal update password: %v", err)})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password berhasil diubah."})
}

