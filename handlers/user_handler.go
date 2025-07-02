package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/pkg/paseto"
	"Sistem-Manajemen-Karyawan/pkg/utils"
	"Sistem-Manajemen-Karyawan/repository"
)

type UserHandler struct {
	userRepo *repository.UserRepository
}

func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}


func (h *UserHandler) GetUserByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "format ID user tidak valid"})
	}

	claims, ok := c.Locals("user").(*paseto.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "tidak terautentikasi atau klaim token tidak valid"})
	}

	if claims.Role != "admin" && claims.UserID.Hex() != idParam {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "akses ditolak. anda hanya dapat melihat profile anda sendiri."})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	user, err := h.userRepo.FindUserByID(ctx, objID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mendapatkan user: %v", err)})
	}

	user.Password = ""
	return c.Status(fiber.StatusOK).JSON(user)
}

func (h *UserHandler) GetAllUsers(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	search := c.Query("search", "")
	role := c.Query("role", "")

	if page < 1 {
		page = 1
	}

	if limit < 1 || limit > 100 {
		limit = 10
	}

	filter := bson.M{}
	if search != "" {
		filter["$or"] = []bson.M{
			{"name": primitive.Regex{Pattern: search, Options: "i"}},
			{"email": primitive.Regex{Pattern: search, Options: "i"}},
		}
	}
	if role != "" {
		filter["role"] = role
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	users, total, err := h.userRepo.GetAllUsers(ctx, filter, int64(page), int64(limit))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mendapatkan semua user: %v", err)})
	}
	for i := range users {
		users[i].Password = ""
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":  users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "format ID user tidak valid"})
	}

	claims, ok := c.Locals("user").(*paseto.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "tidak terautentikasi atau klaim token tidak valid"})
	}

	// Otorisasi Tahap 1: Admin bisa update user manapun, karyawan hanya bisa update dirinya sendiri.
	// Jika karyawan mencoba update orang lain, langsung ditolak di sini.
	if claims.Role != "admin" && claims.UserID.Hex() != idParam {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "akses ditolak. anda hanya dapat mengupdate profil anda sendiri."})
	}

	var payload models.UserUpdatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body", "details": err.Error()})
	}

	// Lakukan validasi payload secara umum
	if errors := util.ValidateStruct(payload); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": errors})
	}

	updateData := bson.M{}
	
	// Logika Otorisasi Per-Field:
	// Karyawan (role != "admin") hanya boleh mengubah 'photo' dan 'address'.
	// Admin (role == "admin") boleh mengubah semua field.

	if claims.Role != "admin" { // Jika user yang melakukan request ADALAH KARYAWAN
		// Izinkan karyawan mengubah field 'photo'
		if payload.Photo != "" {
			updateData["photo"] = payload.Photo
		}
		// Izinkan karyawan mengubah field 'address'
		if payload.Address != "" {
			updateData["address"] = payload.Address
		}

		// Periksa apakah karyawan mencoba mengubah field yang tidak diizinkan
		// Field yang tidak diizinkan: Name, Email, Position, Department, BaseSalary
		if payload.Name != "" || payload.Email != "" || 
		   payload.Position != "" || payload.Department != "" || payload.BaseSalary != 0 {
			
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "akses ditolak. anda tidak diizinkan mengubah nama, email, posisi, departemen, atau gaji dasar.",
			})
		}
	} else { // Jika user yang melakukan request ADALAH ADMIN
		// Admin diizinkan mengubah semua field yang ada di payload
		if payload.Name != "" {
			updateData["name"] = payload.Name
		}
		if payload.Email != "" {
			updateData["email"] = payload.Email
		}
		if payload.Position != "" { // Admin boleh mengubah posisi
			updateData["position"] = payload.Position
		}
		if payload.Department != "" {
			updateData["department"] = payload.Department
		}
		if payload.BaseSalary != 0 {
			updateData["base_salary"] = payload.BaseSalary
		}
		if payload.Address != "" { // Admin juga boleh mengubah alamat
			updateData["address"] = payload.Address
		}
		if payload.Photo != "" { // Admin juga boleh mengubah foto
			updateData["photo"] = payload.Photo
		}
	}

	// Jika setelah otorisasi, tidak ada field yang tersisa untuk diupdate
	if len(updateData) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tidak ada field yang akan diupdate"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	result, err := h.userRepo.UpdateUser(ctx, objID, updateData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mengupdate user: %v", err)})
	}
	if result.ModifiedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "user tidak ditemukan atau tidak ada perubahan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "user berhasil diupdate"})
}

func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "format ID user tidak valid"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	result, err := h.userRepo.DeleteUser(ctx, objID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal menghapus user: %v", err)})
	}
	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "user tidak ditemukan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "user berhasil dihapus"})
}
