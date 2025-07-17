// file: handlers/work_schedule_handler.go

package handlers

import (
	"Sistem-Manajemen-Karyawan/models"
	util "Sistem-Manajemen-Karyawan/pkg/utils"
	"Sistem-Manajemen-Karyawan/repository"
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

// ======================================================================
// FUNGSI getHolidayMap, getExternalHolidaysForFrontend, dan struct HolidayAPIData
// TELAH DIHAPUS DARI FILE INI KARENA SUDAH PINDAH KE pkg/utils
// ======================================================================

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
// @Summary Get All Work Schedules
// @Description Mengambil semua jadwal kerja dalam rentang tanggal tertentu dengan filter hari libur
// @Tags Work Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param start_date query string true "Tanggal mulai (YYYY-MM-DD)"
// @Param end_date query string true "Tanggal selesai (YYYY-MM-DD)"
// @Success 200 {object} object{data=array} "Daftar jadwal kerja berhasil diambil"
// @Failure 400 {object} object{error=string} "Format tanggal tidak valid"
// @Failure 500 {object} object{error=string} "Gagal mengambil jadwal kerja"
// @Router /work-schedules [get]
func (h *WorkScheduleHandler) GetAllWorkSchedules(c *fiber.Ctx) error {
	layout := "2006-01-02"
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate, err := time.Parse(layout, startDateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start_date format"})
	}
	endDate, err := time.Parse(layout, endDateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end_date format"})
	}

	scheduleRules, err := h.workScheduleRepo.FindAllWithFilter(bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch schedule rules"})
	}

	// DIUBAH: Memanggil dari utils
	holidayMap, err := util.GetHolidayMap(startDate.Format("2006"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch holidays"})
	}
	if startDate.Year() != endDate.Year() {
		nextYearHolidays, _ := util.GetHolidayMap(endDate.Format("2006")) // DIUBAH
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
			ruleSet := rrule.Set{}
			ruleSet.RRule(rr)
			instances := ruleSet.Between(startDate, endDate, true)
			for _, instance := range instances {
				instanceDateStr := instance.Format(layout)
				if !holidayMap[instanceDateStr] {
					finalSchedules = append(finalSchedules, models.WorkSchedule{
						ID:             rule.ID,
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
