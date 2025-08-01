// file: handlers/work_schedule_handler.go

package handlers

import (
	"Sistem-Manajemen-Karyawan/models"
	util "Sistem-Manajemen-Karyawan/pkg/utils"
	"Sistem-Manajemen-Karyawan/repository"

	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/teambition/rrule-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WorkScheduleHandler struct {
	workScheduleRepo *repository.WorkScheduleRepository
}

func NewWorkScheduleHandler(repo *repository.WorkScheduleRepository) *WorkScheduleHandler {
	return &WorkScheduleHandler{
		workScheduleRepo: repo,
	}
}

// CreateWorkSchedule godoc
// @Summary Create Work Schedule
// @Description Membuat jadwal kerja baru dengan opsi recurrence rule (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param schedule body models.WorkScheduleCreatePayload true "Data jadwal kerja baru"
// @Success 201 {object} object{message=string,data=models.WorkSchedule} "Jadwal kerja berhasil ditambahkan"
// @Failure 400 {object} object{error=string} "Format data tidak valid"
// @Failure 500 {object} object{error=string} "Gagal menyimpan jadwal kerja"
// @Router /admin/work-schedules [post]
func (h *WorkScheduleHandler) CreateWorkSchedule(c *fiber.Ctx) error {
	var payload models.WorkScheduleCreatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid", "details": err.Error()})
	}

	schedule := models.WorkSchedule{
		ID:             primitive.NewObjectID(),
		Date:           strings.TrimSpace(payload.Date),
		StartTime:      strings.TrimSpace(payload.StartTime),
		EndTime:        strings.TrimSpace(payload.EndTime),
		Note:           payload.Note,
		RecurrenceRule: payload.RecurrenceRule,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	createdSchedule, err := h.workScheduleRepo.Create(&schedule)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan jadwal kerja", "details": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Jadwal kerja berhasil ditambahkan", "data": createdSchedule})
}

// GetHolidays godoc
// @Summary Get Holidays
// @Description Mengambil daftar hari libur nasional untuk tahun tertentu
// @Tags Work Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param year query string false "Tahun (default: tahun sekarang)"
// @Success 200 {object} object "Data hari libur berhasil diambil"
// @Failure 500 {object} object{error=string} "Gagal mengambil data hari libur"
// @Router /work-schedules/holidays [get]
func (h *WorkScheduleHandler) GetHolidays(c *fiber.Ctx) error {
	year := c.Query("year")
	if year == "" {
		year = time.Now().Format("2006")
	}

	// Memanggil fungsi dari utils yang sudah kita buat
	holidaysData, err := util.GetExternalHolidaysForFrontend(year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data hari libur", "details": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(holidaysData)
}

// GetWorkScheduleById godoc
// @Summary Get Work Schedule by ID
// @Description Mengambil detail jadwal kerja berdasarkan ID
// @Tags Work Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Work Schedule ID"
// @Success 200 {object} object{data=models.WorkSchedule} "Jadwal kerja berhasil diambil"
// @Failure 400 {object} object{error=string} "ID jadwal kerja tidak valid"
// @Failure 404 {object} object{error=string} "Jadwal kerja tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal mengambil jadwal kerja"
// @Router /work-schedules/{id} [get]
func (h *WorkScheduleHandler) GetWorkScheduleById(c *fiber.Ctx) error {
	scheduleID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(scheduleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID jadwal kerja tidak valid"})
	}

	schedule, err := h.workScheduleRepo.FindByID(objectID)
	if err != nil {
		if err.Error() == "jadwal tidak ditemukan" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Jadwal kerja tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil jadwal kerja", "details": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": schedule})
}

// GetAllWorkSchedules godoc
// @Summary Get All Work Schedules (for Admin) or My Schedules (for Employee)
// @Description Mengambil jadwal kerja. Admin melihat semua, karyawan melihat jadwal pribadinya.
// @Tags Work Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param start_date query string true "Tanggal mulai (YYYY-MM-DD)"
// @Param end_date query string true "Tanggal selesai (YYYY-MM-DD)"
// @Success 200 {object} object{data=[]models.WorkSchedule} "Daftar jadwal kerja berhasil diambil"
// @Failure 400 {object} object{error=string} "Format tanggal tidak valid"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi atau token tidak valid"
// @Failure 500 {object} object{error=string} "Gagal mengambil jadwal kerja"
// @Router /work-schedules [get]
func (h *WorkScheduleHandler) GetAllWorkSchedules(c *fiber.Ctx) error {
	// Ambil dan validasi parameter tanggal (tidak ada perubahan di sini)
	layout := "2006-01-02"
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate, err := time.Parse(layout, startDateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format start_date tidak valid, gunakan YYYY-MM-DD"})
	}
	endDate, err := time.Parse(layout, endDateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format end_date tidak valid, gunakan YYYY-MM-DD"})
	}

	// ================== PERBAIKAN UTAMA DI SINI ==================
	// 1. Baca data dari c.Locals sebagai tipe data yang benar (*paseto.Claims)
	claims, ok := c.Locals("user").(*models.Claims) // Menggunakan models.Claims sebagai tipe data yang benar
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tipe data token di context tidak valid"})
	}

	// 2. Ekstrak ID dan Role dari claims
	role := claims.Role
	userID := claims.UserID // Menggunakan UserID dari models.Claims
	// ==============================================================

	// Logika kondisional di bawah ini sekarang akan bekerja dengan benar
	if role == "admin" {
		// --- LOGIKA LAMA UNTUK ADMIN (MELIHAT SEMUA) ---
		// Kode ini dipertahankan agar admin tetap bisa melihat semua jadwal tanpa filter.
		scheduleRules, err := h.workScheduleRepo.FindAllWithFilter(bson.M{})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil aturan jadwal"})
		}

		holidayMap, err := util.GetHolidayMap(startDate.Format("2006"))
		if err != nil {
			// Sebaiknya tidak menghentikan proses jika gagal mengambil hari libur, cukup beri peringatan
			fmt.Printf("Peringatan: Gagal mengambil data hari libur: %v\n", err)
			holidayMap = make(map[string]bool) // Inisialisasi map kosong
		}
		if startDate.Year() != endDate.Year() {
			nextYearHolidays, _ := util.GetHolidayMap(endDate.Format("2006"))
			for date, val := range nextYearHolidays {
				holidayMap[date] = val
			}
		}

		finalSchedules := []models.WorkSchedule{}

		for _, rule := range scheduleRules {
			if rule.RecurrenceRule != "" {
				rOption, err := rrule.StrToROption(rule.RecurrenceRule)
				if err != nil {
					continue
				}
				ruleStartDate, _ := time.Parse(layout, rule.Date)
				rOption.Dtstart = ruleStartDate
				rr, err := rrule.NewRRule(*rOption)
				if err != nil {
					continue
				}
				instances := rr.Between(startDate, endDate, true)
				for _, instance := range instances {
					instanceDateStr := instance.Format(layout)
					if !holidayMap[instanceDateStr] {
						finalSchedules = append(finalSchedules, models.WorkSchedule{
							ID:             rule.ID,
							UserID:         rule.UserID,
							Date:           instanceDateStr,
							StartTime:      rule.StartTime,
							EndTime:        rule.EndTime,
							Note:           rule.Note,
							RecurrenceRule: rule.RecurrenceRule,
						})
					}
				}
			} else {
				ruleDate, _ := time.Parse(layout, rule.Date)
				if (ruleDate.After(startDate) || ruleDate.Equal(startDate)) && (ruleDate.Before(endDate) || ruleDate.Equal(endDate)) {
					if !holidayMap[rule.Date] {
						finalSchedules = append(finalSchedules, rule)
					}
				}
			}
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": finalSchedules})

	} else {
		// --- LOGIKA BARU DAN EFISIEN UNTUK KARYAWAN ---
		var dailySchedules []models.WorkSchedule
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format(layout)
			// Gunakan 'userID' yang sudah kita dapatkan dari token
			schedule, err := h.workScheduleRepo.FindApplicableScheduleForUser(c.Context(), userID, dateStr)
			if err == nil && schedule != nil {
				dailySchedules = append(dailySchedules, *schedule)
			}
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": dailySchedules})
	}
}

// UpdateWorkSchedule godoc
// @Summary Update Work Schedule
// @Description Memperbarui jadwal kerja berdasarkan ID (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Work Schedule ID"
// @Param schedule body models.WorkScheduleUpdatePayload true "Data update jadwal kerja"
// @Success 200 {object} object{message=string} "Jadwal kerja berhasil diperbarui"
// @Failure 400 {object} object{error=string} "ID tidak valid atau validasi gagal"
// @Failure 404 {object} object{error=string} "Jadwal tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal memperbarui jadwal kerja"
// @Router /admin/work-schedules/{id} [put]
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

	err = h.workScheduleRepo.UpdateByID(objectID, &payload)
	if err != nil {
		if strings.Contains(err.Error(), "jadwal tidak ditemukan") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Jadwal tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui jadwal kerja", "details": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Jadwal kerja berhasil diperbarui"})
}

// DeleteWorkSchedule godoc
// @Summary Delete Work Schedule
// @Description Menghapus jadwal kerja berdasarkan ID (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Work Schedule ID"
// @Success 200 {object} object{message=string} "Jadwal kerja berhasil dihapus"
// @Failure 400 {object} object{error=string} "ID jadwal kerja tidak valid"
// @Failure 404 {object} object{error=string} "Jadwal tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal menghapus jadwal kerja"
// @Router /admin/work-schedules/{id} [delete]
func (h *WorkScheduleHandler) DeleteWorkSchedule(c *fiber.Ctx) error {
	scheduleID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(scheduleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID jadwal kerja tidak valid"})
	}

	err = h.workScheduleRepo.DeleteByID(objectID)
	if err != nil {
		if strings.Contains(err.Error(), "jadwal tidak ditemukan") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Jadwal tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus jadwal kerja", "details": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Jadwal kerja berhasil dihapus"})
}
