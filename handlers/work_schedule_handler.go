package handlers

import (

	"strings"
	"time"

	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/repository" // Pastikan package repository diimpor

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WorkScheduleHandler memiliki ketergantungan pada WorkScheduleRepository
type WorkScheduleHandler struct {
	workScheduleRepo *repository.WorkScheduleRepository
}

// NewWorkScheduleHandler membuat instance baru dari WorkScheduleHandler
func NewWorkScheduleHandler(repo *repository.WorkScheduleRepository) *WorkScheduleHandler {
	return &WorkScheduleHandler{
		workScheduleRepo: repo,
	}
}

// Buat jadwal kerja baru (Admin Only)
// Ini sekarang adalah method dari WorkScheduleHandler
func (h *WorkScheduleHandler) CreateWorkSchedule(c *fiber.Ctx) error {
	var payload models.WorkScheduleCreatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid", "details": err.Error()})
	}

	userID, err := primitive.ObjectIDFromHex(payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "UserID tidak valid"})
	}

	schedule := models.WorkSchedule{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Date:      strings.TrimSpace(payload.Date),
		StartTime: strings.TrimSpace(payload.StartTime),
		EndTime:   strings.TrimSpace(payload.EndTime),
		Note:      payload.Note,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Menggunakan repository untuk operasi database
	createdSchedule, err := h.workScheduleRepo.Create(&schedule)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan jadwal kerja", "details": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Jadwal kerja berhasil ditambahkan", "data": createdSchedule})
}

// GetAllWorkSchedules sekarang adalah method dari WorkScheduleHandler
func (h *WorkScheduleHandler) GetAllWorkSchedules(c *fiber.Ctx) error {
	filter := bson.M{}

	if userID := c.Query("user_id"); userID != "" {
		uid, err := primitive.ObjectIDFromHex(userID)
		if err == nil {
			filter["user_id"] = uid
		}
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	if startDate != "" && endDate != "" {
		filter["date"] = bson.M{
			"$gte": startDate,
			"$lte": endDate,
		}
	} else if date := c.Query("date"); date != "" {
		filter["date"] = date
	}

	// Menggunakan repository untuk operasi database
	// Karena repository yang Anda berikan tidak memiliki method FindAll dengan filter dinamis,
	// kita akan sedikit menyesuaikan atau menambahkan di repository. Untuk saat ini,
	// saya asumsikan ada atau kita akan menggunakan query langsung yang bisa ditangani repo.
	// Paling ideal, Anda tambahkan method `FindAll(filter bson.M)` di `WorkScheduleRepository`.
	// Jika tidak, Anda bisa langsung menggunakan `collection.Find` seperti sebelumnya.
	// Untuk demo ini, saya akan tetap menggunakan `FindAllWithFilter` yang saya asumsikan ada di repo.
	// Jika repo Anda belum punya, ini perlu ditambahkan.
	schedules, err := h.workScheduleRepo.FindAllWithFilter(filter) // Asumsi method ini ada di repo
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data jadwal kerja", "details": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": schedules})
}

// Lihat jadwal kerja saya (untuk Karyawan yang sedang login)
// Ini sekarang adalah method dari WorkScheduleHandler
func (h *WorkScheduleHandler) GetMyWorkSchedules(c *fiber.Ctx) error {
	user := c.Locals("user")
	claims, ok := user.(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User tidak ditemukan dalam konteks"})
	}

	// Menggunakan repository untuk operasi database
	schedules, err := h.workScheduleRepo.FindByUser(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil jadwal kerja"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": schedules})
}

// UpdateWorkSchedule sekarang adalah method dari WorkScheduleHandler
func (h *WorkScheduleHandler) UpdateWorkSchedule(c *fiber.Ctx) error {
	scheduleID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(scheduleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID jadwal kerja tidak valid"})
	}

	var payload models.WorkScheduleUpdatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid", "details": err.Error()})
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validasi gagal: " + err.Error()})
	}

	// Menggunakan repository untuk operasi database
	err = h.workScheduleRepo.UpdateByID(objectID, &payload)
	if err != nil {
		// Cek apakah error karena tidak ditemukan (sesuaikan pesan error dari repo Anda)
		if strings.Contains(err.Error(), "jadwal tidak ditemukan") { // Pesan error dari repository.DeleteByID
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Jadwal tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui jadwal kerja", "details": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Jadwal kerja berhasil diperbarui"})
}

// DeleteWorkSchedule sekarang adalah method dari WorkScheduleHandler
func (h *WorkScheduleHandler) DeleteWorkSchedule(c *fiber.Ctx) error {
	scheduleID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(scheduleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID jadwal kerja tidak valid"})
	}

	// Menggunakan repository untuk operasi database
	err = h.workScheduleRepo.DeleteByID(objectID)
	if err != nil {
		// Cek apakah error karena tidak ditemukan (sesuaikan pesan error dari repo Anda)
		if strings.Contains(err.Error(), "jadwal tidak ditemukan") { // Pesan error dari repository.DeleteByID
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Jadwal kerja tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus jadwal kerja", "details": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Jadwal kerja berhasil dihapus"})
}