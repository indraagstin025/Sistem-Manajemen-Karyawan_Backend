package handlers

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"Sistem-Manajemen-Karyawan/models"
	util "Sistem-Manajemen-Karyawan/pkg/utils"
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
// @Success 200 {object} models.User "User berhasil ditemukan"
// @Failure 400 {object} object{error=string} "Invalid user ID format"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak - hanya bisa melihat data sendiri"
// @Failure 404 {object} object{error=string} "User tidak ditemukan"
// @Failure 500 {object} object{error=string} "Internal server error"
// @Router /users/{id} [get]
func (h *UserHandler) GetUserByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "format ID user tidak valid"})
	}

	claims, ok := c.Locals("user").(*models.Claims)
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
		log.Printf("Error getting user by ID: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mendapatkan user: %v", err)})
	}

	user.Password = ""
	return c.Status(fiber.StatusOK).JSON(user)
}

// GetAllUsers godoc
// @Summary Get All Users
// @Description Mendapatkan semua data users dengan pagination dan filter (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Param search query string false "Search by name or email"
// @Param role query string false "Filter by role"
// @Success 200 {object} object{data=array,total=int,page=int,limit=int} "Data users berhasil diambil"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak - hanya admin"
// @Failure 500 {object} object{error=string} "Gagal mengambil data users"
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
// @Success 200 {object} object{message=string} "User berhasil diupdate"
// @Failure 400 {object} object{error=string,errors=array} "Invalid request body, user ID, atau validation error"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak - hanya bisa update data sendiri"
// @Failure 404 {object} object{message=string} "User tidak ditemukan"
// @Failure 500 {object} object{error=string} "Internal server error"
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "format ID user tidak valid"})
	}

	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "tidak terautentikasi atau klaim token tidak valid"})
	}

	if claims.Role != "admin" && claims.UserID.Hex() != idParam {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "akses ditolak. anda hanya dapat mengupdate profil anda sendiri."})
	}

	var payload models.UserUpdatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body", "details": err.Error()})
	}

	if errors := util.ValidateStruct(payload); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": errors})
	}

	updateData := bson.M{}

	if claims.Role != "admin" {
		if payload.Photo != "" {
			updateData["photo"] = payload.Photo
		}
		if payload.Address != "" {
			updateData["address"] = payload.Address
		}

		if payload.Name != "" || payload.Email != "" ||
			payload.Position != "" || payload.Department != "" || payload.BaseSalary != 0 {

			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "akses ditolak. anda tidak diizinkan mengubah nama, email, posisi, departemen, atau gaji dasar.",
			})
		}
	} else {
		if payload.Name != "" {
			updateData["name"] = payload.Name
		}
		if payload.Email != "" {
			updateData["email"] = payload.Email
		}
		if payload.Position != "" {
			updateData["position"] = payload.Position
		}
		if payload.Department != "" {
			updateData["department"] = payload.Department
		}
		if payload.BaseSalary != 0 {
			updateData["base_salary"] = payload.BaseSalary
		}
		if payload.Address != "" {
			updateData["address"] = payload.Address
		}
		if payload.Photo != "" {
			updateData["photo"] = payload.Photo
		}
	}

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

// DeleteUser godoc
// @Summary Delete User
// @Description Menghapus user berdasarkan ID (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} object{message=string} "User berhasil dihapus"
// @Failure 400 {object} object{error=string} "Invalid ID format"
// @Failure 404 {object} object{message=string} "User tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal menghapus user"
// @Router /admin/users/{id} [delete]
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

// GetDashboardStats godoc
// @Summary Get Dashboard Statistics
// @Description Mendapatkan berbagai statistik untuk dashboard admin (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.DashboardStats "Statistik dashboard berhasil diambil"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak - hanya admin"
// @Failure 500 {object} object{error=string} "Gagal mengambil statistik dashboard"
// @Router /admin/dashboard-stats [get]
func (h *UserHandler) GetDashboardStats(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	stats, err := h.userRepo.GetDashboardStats(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mengambil statistik dashboard: %v", err)})
	}
	return c.Status(fiber.StatusOK).JSON(stats)
}

// UploadProfilePhoto godoc
// @Summary Upload User Profile Photo
// @Description Mengunggah foto profil untuk user tertentu. Hanya admin atau user itu sendiri yang bisa mengunggah.
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param photo formData file true "File foto profil (JPG, PNG, GIF, WEBP, maks 5MB)"
// @Success 200 {object} object{message=string,photo_url=string} "Foto profil berhasil diunggah"
// @Failure 400 {object} object{error=string} "Invalid file format, file size, atau no file uploaded"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak"
// @Failure 404 {object} object{message=string} "User tidak ditemukan"
// @Failure 500 {object} object{error=string} "Internal server error"
// @Router /users/{id}/upload-photo [post]
func (h *UserHandler) UploadProfilePhoto(c *fiber.Ctx) error {
	userID := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format ID user tidak valid"})
	}

	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tidak terautentikasi atau klaim token tidak valid"})
	}

	if claims.Role != "admin" && claims.UserID.Hex() != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak. Anda hanya dapat mengunggah foto profil Anda sendiri."})
	}

	file, err := c.FormFile("photo")
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tidak ada file foto yang diunggah."})
		}

		log.Printf("Error mengambil file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil file."})
	}

	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !allowedTypes[file.Header.Get("Content-Type")] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format file tidak didukung. Hanya JPG, PNG, GIF, WEBP yang diizinkan."})
	}

	const maxFileSize = 5 * 1024 * 1024
	if file.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Ukuran file terlalu besar. Maksimal %d MB.", maxFileSize/1024/1024)})
	}

	uploadDir := "./uploads"
	fileName := fmt.Sprintf("%s_%d%s", userID, time.Now().Unix(), filepath.Ext(file.Filename))
	filePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveFile(file, filePath); err != nil {
		log.Printf("Error menyimpan file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file foto."})
	}

	photoURL := fmt.Sprintf("http://localhost:3000/uploads/%s", fileName)

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	updateData := bson.M{"photo": photoURL}
	result, err := h.userRepo.UpdateUser(ctx, objID, updateData)
	if err != nil {
		log.Printf("Error mengupdate URL foto di database: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui URL foto di database."})
	}
	if result.ModifiedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User tidak ditemukan atau foto tidak berubah."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Foto profil berhasil diunggah.",
		"photo_url": photoURL,
	})
}
