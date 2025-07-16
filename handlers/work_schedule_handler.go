package handlers

import (
    "Sistem-Manajemen-Karyawan/models"
    "Sistem-Manajemen-Karyawan/repository"
    "encoding/json"
    "io/ioutil"
    "net/http"
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

// ## CreateWorkSchedule Diperbarui untuk Menyimpan Aturan Perulangan ##
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
        RecurrenceRule: payload.RecurrenceRule, // Menyimpan aturan perulangan
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }

    createdSchedule, err := h.workScheduleRepo.Create(&schedule)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan jadwal kerja", "details": err.Error()})
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Jadwal kerja berhasil ditambahkan", "data": createdSchedule})
}

type HolidayAPIData struct {
	Date              string `json:"holiday_date"`
	Name              string `json:"holiday_name"`
	IsNationalHoliday bool `json:"is_national_holiday"`
}

// Fungsi pembantu untuk mengambil hari libur dari API eksternal dan mengembalikan slice models.Holiday
func getExternalHolidaysForFrontend(year string) ([]models.Holiday, error) {
	resp, err := http.Get("https://api-harilibur.vercel.app/api?year=" + year)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawHolidays []HolidayAPIData
	if err := json.Unmarshal(body, &rawHolidays); err != nil {
		return nil, err
	}

	var holidays []models.Holiday
	for _, rawHoliday := range rawHolidays {
		if rawHoliday.IsNationalHoliday {
			holidays = append(holidays, models.Holiday{
				Date: rawHoliday.Date,
				Name: rawHoliday.Name,
			})
		}
	}
	return holidays, nil
}

// Fungsi pembantu untuk mengambil dan memetakan hari libur agar mudah dicek (untuk filter internal)
func getHolidayMap(year string) (map[string]bool, error) {
	holidayMap := make(map[string]bool)
	resp, err := http.Get("https://api-harilibur.vercel.app/api?year=" + year)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawHolidays []HolidayAPIData
	if err := json.Unmarshal(body, &rawHolidays); err != nil {
		return nil, err
	}
	for _, rawHoliday := range rawHolidays {
		if rawHoliday.IsNationalHoliday {
			holidayMap[rawHoliday.Date] = true
		}
	}

	return holidayMap, nil
}

// GetHolidays adalah handler baru untuk melayani data hari libur ke frontend
func (h *WorkScheduleHandler) GetHolidays(c *fiber.Ctx) error {
	year := c.Query("year")
	if year == "" {
		year = time.Now().Format("2006")
	}

	holidaysData, err := getExternalHolidaysForFrontend(year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data hari libur", "details": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(holidaysData)
}


// GetWorkScheduleById mendapatkan satu aturan jadwal kerja berdasarkan ID
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

// ## GetAllWorkSchedules Dirombak Total untuk Logika Otomatis ##
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

	holidayMap, err := getHolidayMap(startDate.Format("2006"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch holidays"})
	}
	if startDate.Year() != endDate.Year() {
		nextYearHolidays, _ := getHolidayMap(endDate.Format("2006"))
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

// Fungsi Update dan Delete tetap ada, berguna untuk mengubah/menghapus aturan perulangan itu sendiri.
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

func (h *WorkScheduleHandler) DeleteWorkSchedule(c *fiber.Ctx) error {
	scheduleID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(scheduleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID jadwal kerja tidak valid"})
	}

	err = h.workScheduleRepo.DeleteByID(objectID)
	if err != nil {
		if strings.Contains(err.Error(), "jadwal tidak ditemukan") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Jadwal kerja tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus jadwal kerja", "details": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Jadwal kerja berhasil dihapus"})
}