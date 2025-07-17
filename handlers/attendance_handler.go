package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/repository"
)

type AttendanceHandler struct {
	repo             repository.AttendanceRepository
	workScheduleRepo *repository.WorkScheduleRepository
}

func NewAttendanceHandler(repo repository.AttendanceRepository, workScheduleRepo *repository.WorkScheduleRepository) *AttendanceHandler {
	return &AttendanceHandler{
		repo:             repo,
		workScheduleRepo: workScheduleRepo,
	}

}

// ScanQRCode godoc
// @Summary Scan QR Code untuk Check-in/Check-out
// @Description Melakukan scan QR code untuk proses check-in atau check-out karyawan
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body models.QRCodeScanPayload true "Data QR Code scan"
// @Success 200 {object} object{message=string} "Berhasil check-in/check-out"
// @Success 201 {object} object{message=string} "Berhasil check-in"
// @Failure 400 {object} object{error=string} "Payload tidak valid atau QR Code bermasalah"
// @Failure 404 {object} object{error=string} "QR Code tidak ditemukan"
// @Failure 409 {object} object{error=string} "Sudah melakukan check-in dan check-out"
// @Failure 500 {object} object{error=string} "Gagal melakukan check-in/check-out"
// @Router /attendance/scan [post]
// handlers/attendance_handler.go
// file: handlers/attendance_handler.go
func (h *AttendanceHandler) ScanQRCode(c *fiber.Ctx) error {
	var payload models.QRCodeScanPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payload tidak valid: " + err.Error()})
	}

	wib, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(wib)
	today := now.Format("2006-01-02")

	// 1. Validasi QR Code
	qrCode, err := h.repo.FindQRCodeByValue(c.Context(), payload.QRCodeValue)
	if err != nil || qrCode == nil || qrCode.Date != today || now.After(qrCode.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "QR Code tidak valid atau sudah kadaluarsa."})
	}

	userID, err := primitive.ObjectIDFromHex(payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format User ID tidak valid."})
	}

	// 2. Cek duplikasi absensi
	existingAttendance, err := h.repo.FindAttendanceByUserAndDate(c.Context(), userID, today)
	if err == nil && existingAttendance != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": fmt.Sprintf("Anda sudah memiliki record absensi untuk hari ini dengan status: %s.", existingAttendance.Status)})
	}

	// 3. Panggil fungsi repository. Sekarang tipenya sudah benar.
	todaysSchedule, err := h.workScheduleRepo.FindApplicableScheduleForUser(c.Context(), userID, today)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	// -- BARIS TYPE ASSERTION YANG KEMARIN KITA TAMBAHKAN, SEKARANG DIHAPUS --

	// 4. Logika perbandingan waktu
	scheduledStartTime, _ := time.ParseInLocation("15:04", todaysSchedule.StartTime, wib)
	scheduleCheckInTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		scheduledStartTime.Hour(), scheduledStartTime.Minute(), 0, 0, wib,
	)

	gracePeriod := 30 * time.Minute
	latestCheckInTime := scheduleCheckInTime.Add(gracePeriod)

	var attendanceStatus string
	if now.After(latestCheckInTime) {
		attendanceStatus = "Terlambat"
	} else {
		attendanceStatus = "Hadir"
	}

	// 5. Membuat record absensi baru
	newAttendance := models.Attendance{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Date:      today,
		CheckIn:   now.Format("15:04"),
		Status:    attendanceStatus,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = h.repo.CreateAttendance(c.Context(), &newAttendance)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan data check-in: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": fmt.Sprintf("Berhasil check-in pukul %s. Status Anda: %s", newAttendance.CheckIn, newAttendance.Status),
	})
}

// GenerateQRCode godoc
// @Summary Generate QR Code untuk Attendance
// @Description Membuat QR code baru untuk attendance atau mengembalikan QR code yang masih aktif
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{message=string,qr_code_image=string,expires_at=string,qr_code_value=string} "QR Code berhasil dibuat atau sudah ada"
// @Failure 500 {object} object{error=string} "Gagal membuat QR Code"
// @Router /attendance/generate-qr [get]
func (h *AttendanceHandler) GenerateQRCode(c *fiber.Ctx) error {
	const QR_CODE_DURATION = 30 * time.Second

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	wib, _ := time.LoadLocation("Asia/Jakarta")
	currentTimeInWIB := time.Now().In(wib)
	todayStr := currentTimeInWIB.Format("2006-01-02")

	existingQRCode, err := h.repo.FindActiveQRCodeByDate(ctx, todayStr)

	if err == nil && existingQRCode != nil && currentTimeInWIB.Before(existingQRCode.ExpiresAt) {

		png, err := qrcode.Encode(existingQRCode.Code, qrcode.Medium, 256)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal re-encode gambar QR Code yang sudah ada."})
		}
		encodedString := base64.StdEncoding.EncodeToString(png)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":       "QR Code aktif untuk hari ini sudah ada.",
			"qr_code_image": "data:image/png;base64," + encodedString,
			"expires_at":    existingQRCode.ExpiresAt,
			"qr_code_value": existingQRCode.Code,
		})
	}

	uniqueCode := uuid.New().String()
	expiresAt := currentTimeInWIB.Add(QR_CODE_DURATION)

	newQRCode := &models.QRCode{
		ID:        primitive.NewObjectID(),
		Code:      uniqueCode,
		Date:      todayStr,
		ExpiresAt: expiresAt,
		CreatedAt: currentTimeInWIB,
		UpdatedAt: currentTimeInWIB,
	}

	_, err = h.repo.CreateQRCode(ctx, newQRCode)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan data QR Code baru: " + err.Error()})
	}

	png, err := qrcode.Encode(uniqueCode, qrcode.Medium, 256)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat gambar QR Code baru: " + err.Error()})
	}

	encodedString := base64.StdEncoding.EncodeToString(png)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":       "QR Code baru berhasil dibuat",
		"qr_code_image": "data:image/png;base64," + encodedString,
		"expires_at":    expiresAt,
		"qr_code_value": uniqueCode,
	})
}

// Di attendance_handler.go

// GetAttendanceHistoryForAdmin godoc
// @Summary Get Attendance History for All Employees (Admin)
// @Description Mengambil riwayat kehadiran semua karyawan dengan filter dan pagination (admin only)
// @Tags Admin Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Param user_id query string false "Filter by User ID"
// @Param start_date query string false "Filter by Start Date (YYYY-MM-DD)"
// @Param end_date query string false "Filter by End Date (YYYY-MM-DD)"
// @Success 200 {object} object{data=array,total=int,page=int,limit=int} "Riwayat kehadiran berhasil diambil"
// @Failure 400 {object} object{error=string} "Invalid parameters"
// @Failure 401 {object} object{error=string} "Unauthorized"
// @Failure 403 {object} object{error=string} "Forbidden"
// @Failure 500 {object} object{error=string} "Internal server error"
// @Router /admin/attendance/history [get]
func (h *AttendanceHandler) GetAttendanceHistoryForAdmin(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok || claims.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak. Hanya admin."})
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	userIDParam := c.Query("user_id", "")
	startDateStr := c.Query("start_date", "")
	endDateStr := c.Query("end_date", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	filter := bson.M{}
	if userIDParam != "" {
		objID, err := primitive.ObjectIDFromHex(userIDParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format User ID tidak valid."})
		}
		filter["user_id"] = objID
	}

	if startDateStr != "" && endDateStr != "" {

		filter["date"] = bson.M{
			"$gte": startDateStr,
			"$lte": endDateStr,
		}
	} else if startDateStr != "" {
		filter["date"] = bson.M{"$gte": startDateStr}
	} else if endDateStr != "" {
		filter["date"] = bson.M{"$lte": endDateStr}
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	attendances, total, err := h.repo.GetAllAttendancesWithUserDetails(ctx, filter, int64(page), int64(limit))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil riwayat kehadiran: " + err.Error()})
	}

	if attendances == nil {
		attendances = []models.AttendanceWithUser{}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":  attendances,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// Tambahkan fungsi ini di attendance_handler.go
// GetMyTodayAttendance godoc
// @Summary Get My Today's Attendance
// @Description Mengambil data absensi hari ini untuk user yang sedang login
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.Attendance "Data absensi hari ini berhasil diambil"
// @Success 200 {object} object "Null jika belum ada absensi hari ini"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Router /attendance/my-today [get]
func (h *AttendanceHandler) GetMyTodayAttendance(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tidak terautentikasi"})
	}

	wib, _ := time.LoadLocation("Asia/Jakarta")
	today := time.Now().In(wib).Format("2006-01-02")

	attendance, err := h.repo.FindAttendanceByUserAndDate(c.Context(), claims.UserID, today)
	if err != nil {
		return c.Status(fiber.StatusOK).JSON(nil)
	}

	return c.Status(fiber.StatusOK).JSON(attendance)
}

// GetTodayAttendance godoc
// @Summary Get Today's Attendance List
// @Description Mengambil daftar kehadiran karyawan untuk hari ini dengan detail user
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.AttendanceWithUser "Daftar kehadiran hari ini berhasil diambil"
// @Failure 500 {object} object{error=string} "Gagal mengambil daftar kehadiran"
// @Router /attendance/today [get]
func (h *AttendanceHandler) GetTodayAttendance(c *fiber.Ctx) error {
	// --- GetTodayAttendanceWithUserDetails sekarang menerima context ---
	attendanceList, err := h.repo.GetTodayAttendanceWithUserDetails(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil daftar kehadiran: " + err.Error(),
		})
	}

	// Jika list kosong, kembalikan slice kosong (bukan nil)
	if attendanceList == nil {
		return c.Status(fiber.StatusOK).JSON([]models.AttendanceWithUser{})
	}

	return c.Status(fiber.StatusOK).JSON(attendanceList)
}

// GetMyAttendanceHistory godoc
// @Summary Get My Attendance History
// @Description Mengambil seluruh riwayat absensi untuk user yang sedang login
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Attendance "Riwayat absensi berhasil diambil"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi atau token tidak valid"
// @Failure 500 {object} object{error=string} "Gagal mengambil riwayat absensi"
// @Router /attendance/my-history [get]
func (h *AttendanceHandler) GetMyAttendanceHistory(c *fiber.Ctx) error {
	// Ambil claims dari token yang sudah divalidasi oleh AuthMiddleware
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Tidak terautentikasi atau data sesi rusak",
		})
	}

	userID := claims.UserID

	attendanceHistory, err := h.repo.FindAttendanceByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil riwayat kehadiran: " + err.Error(),
		})
	}

	if attendanceHistory == nil {
		return c.Status(fiber.StatusOK).JSON([]models.Attendance{})
	}

	return c.Status(fiber.StatusOK).JSON(attendanceHistory)
}
