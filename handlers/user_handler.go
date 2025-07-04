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

// GetUserByID godoc
// @Summary Get User by ID
// @Description Mendapatkan detail user berdasarkan ID (user hanya bisa melihat data diri sendiri, admin bisa melihat semua)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} models.GetUserSuccessResponse "User berhasil ditemukan"
// @Failure 400 {object} models.ErrorResponse "Invalid user ID format"
// @Failure 401 {object} models.UnauthorizedErrorResponse "Tidak terautentikasi"
// @Failure 403 {object} models.ForbiddenErrorResponse "Akses ditolak - hanya bisa melihat data sendiri"
// @Failure 404 {object} models.NotFoundErrorResponse "User tidak ditemukan"
// @Failure 500 {object} models.ErrorResponse "Gagal mengambil data user"
// @Router /users/{id} [get]
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

// GetAllUsers godoc
// @Summary Get All Users (Admin Only)
// @Description Mendapatkan semua data users - hanya admin yang dapat mengakses endpoint ini
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.GetAllUsersSuccessResponse "Data users berhasil diambil"
// @Failure 401 {object} models.UnauthorizedErrorResponse "Tidak terautentikasi"
// @Failure 403 {object} models.ForbiddenErrorResponse "Akses ditolak - hanya admin"
// @Failure 500 {object} models.ErrorResponse "Gagal mengambil data users"
// @Router /admin/users [get]
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

// UpdateUser godoc
// @Summary Update User
// @Description Update data user (user hanya bisa update data diri sendiri, admin bisa update semua)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param user body models.UserUpdatePayload true "Data update user"
// @Success 200 {object} models.UpdateUserSuccessResponse "User berhasil diupdate"
// @Failure 400 {object} models.ErrorResponse "Invalid request body atau user ID"
// @Failure 401 {object} models.UnauthorizedErrorResponse "Tidak terautentikasi"
// @Failure 403 {object} models.ForbiddenErrorResponse "Akses ditolak - hanya bisa update data sendiri"
// @Failure 404 {object} models.NotFoundErrorResponse "User tidak ditemukan"
// @Failure 500 {object} models.ErrorResponse "Gagal update user"
// @Router /users/{id} [put]
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
