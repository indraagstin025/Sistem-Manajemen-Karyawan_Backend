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
	"Sistem-Manajemen-Karyawan/repository"
	"Sistem-Manajemen-Karyawan/pkg/utils" // Menggunakan alias 'utils' sesuai impor Anda
)

// DepartmentHandler menyediakan metode untuk berinteraksi dengan departemen.
type DepartmentHandler struct {
	deptRepo repository.DepartmentRepository
	// HAPUS: validate *validator.Validate // Tidak perlu lagi karena validator global di pkg/utils
}

// NewDepartmentHandler membuat instance DepartmentHandler baru.
func NewDepartmentHandler(deptRepo repository.DepartmentRepository) *DepartmentHandler {
	return &DepartmentHandler{
		deptRepo: deptRepo,
		// HAPUS: validate: utils.NewValidator(), // Tidak perlu lagi
	}
}

// CreateDepartment godoc
// @Summary Create Department
// @Description Menambahkan departemen baru (hanya admin)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param department body models.Department true "Data departemen baru"
// @Success 201 {object} fiber.Map "message: Departemen berhasil ditambahkan"
// @Failure 400 {object} fiber.Map "error: Invalid request body"
// @Failure 409 {object} fiber.Map "error: Nama departemen sudah ada"
// @Failure 500 {object} fiber.Map "error: Gagal membuat departemen"
// @Router /admin/departments [post]
func (h *DepartmentHandler) CreateDepartment(c *fiber.Ctx) error {
    var newDept models.Department
    if err := c.BodyParser(&newDept); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
    }

    // NEW: Gunakan langsung utils.ValidateStruct
    if errors := util.ValidateStruct(newDept); errors != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": errors}) // Mengembalikan slice errors langsung
    }

    ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
    defer cancel()

    // Cek apakah nama departemen sudah ada
    existingDept, err := h.deptRepo.FindDepartmentByName(ctx, newDept.Name)
    if err != nil && err.Error() != "departemen tidak ditemukan" {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal memeriksa departemen: %v", err)})
    }
    if existingDept != nil {
        return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Nama departemen sudah ada"})
    }

    result, err := h.deptRepo.CreateDepartment(ctx, &newDept)
    if err != nil {
        if err.Error() == "nama departemen sudah ada" {
            return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Nama departemen sudah ada"})
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal membuat departemen: %v", err)})
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "message": "Departemen berhasil ditambahkan",
        "id":      result.InsertedID,
    })
}

// GetAllDepartments godoc
// @Summary Get All Departments
// @Description Mendapatkan daftar semua departemen
// @Tags Departments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Department "Daftar departemen berhasil diambil"
// @Failure 500 {object} fiber.Map "error: Gagal mengambil departemen"
// @Router /departments [get]
func (h *DepartmentHandler) GetAllDepartments(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	departments, err := h.deptRepo.GetAllDepartments(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal mengambil departemen: %v", err)})
	}
	return c.Status(fiber.StatusOK).JSON(departments)
}

// GetDepartmentByID godoc
// @Summary Get Department by ID
// @Description Mendapatkan detail departemen berdasarkan ID
// @Tags Departments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Department ID"
// @Success 200 {object} models.Department "Departemen berhasil ditemukan"
// @Failure 400 {object} fiber.Map "error: Invalid ID format"
// @Failure 404 {object} fiber.Map "error: Departemen tidak ditemukan"
// @Failure 500 {object} fiber.Map "error: Gagal mengambil departemen"
// @Router /departments/{id} [get]
func (h *DepartmentHandler) GetDepartmentByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format ID departemen tidak valid"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	dept, err := h.deptRepo.GetDepartmentByID(ctx, objID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Departemen tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal mengambil departemen: %v", err)})
	}
	return c.Status(fiber.StatusOK).JSON(dept)
}

// UpdateDepartment godoc
// @Summary Update Department
// @Description Memperbarui departemen berdasarkan ID (hanya admin)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Department ID"
// @Param department body models.Department true "Data departemen untuk diupdate"
// @Success 200 {object} fiber.Map "message: Departemen berhasil diupdate"
// @Failure 400 {object} fiber.Map "error: Invalid request body or ID format"
// @Failure 404 {object} fiber.Map "error: Departemen tidak ditemukan"
// @Failure 500 {object} fiber.Map "error: Gagal mengupdate departemen"
// @Router /admin/departments/{id} [put]
func (h *DepartmentHandler) UpdateDepartment(c *fiber.Ctx) error {
    idParam := c.Params("id")
    objID, err := primitive.ObjectIDFromHex(idParam)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format ID departemen tidak valid"})
    }

    var updatePayload models.Department
    if err := c.BodyParser(&updatePayload); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
    }

    // NEW: Gunakan langsung utils.ValidateStruct
    if errors := util.ValidateStruct(updatePayload); errors != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": errors}) // Mengembalikan slice errors langsung
    }

    ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
    defer cancel()

    updateData := bson.M{}
    if updatePayload.Name != "" {
        // Cek duplikasi nama jika nama diubah
        existingDept, err := h.deptRepo.FindDepartmentByName(ctx, updatePayload.Name)
        if err != nil && err.Error() != "departemen tidak ditemukan" {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal memeriksa departemen: %v", err)})
        }
        if existingDept != nil && existingDept.ID != objID { // Jika nama sudah ada dan bukan departemen yang sedang diupdate
            return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Nama departemen sudah ada"})
        }
        updateData["name"] = updatePayload.Name
    }

    if len(updateData) == 0 {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tidak ada data untuk diupdate"})
    }

    result, err := h.deptRepo.UpdateDepartment(ctx, objID, updateData)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal mengupdate departemen: %v", err)})
    }
    if result.ModifiedCount == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Departemen tidak ditemukan atau tidak ada perubahan"})
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Departemen berhasil diupdate"})
}

// DeleteDepartment godoc
// @Summary Delete Department
// @Description Menghapus departemen berdasarkan ID (hanya admin)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Department ID"
// @Success 200 {object} fiber.Map "message: Departemen berhasil dihapus"
// @Failure 400 {object} fiber.Map "error: Invalid ID format"
// @Failure 404 {object} fiber.Map "error: Departemen tidak ditemukan"
// @Failure 500 {object} fiber.Map "error: Gagal menghapus departemen"
// @Router /admin/departments/{id} [delete]
func (h *DepartmentHandler) DeleteDepartment(c *fiber.Ctx) error {
    idParam := c.Params("id")
    objID, err := primitive.ObjectIDFromHex(idParam)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format ID departemen tidak valid"})
    }

    ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
    defer cancel()

    result, err := h.deptRepo.DeleteDepartment(ctx, objID)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal menghapus departemen: %v", err)})
    }
    if result.DeletedCount == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Departemen tidak ditemukan"})
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Departemen berhasil dihapus"})
}
