package handlers

import (
	"context" // Pastikan ini diimpor
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/repository" // Pastikan path ini benar
)

type AttendanceHandler struct {
	repo repository.AttendanceRepository
	workScheduleRepo *repository.WorkScheduleRepository
}

func NewAttendanceHandler(repo repository.AttendanceRepository, workScheduleRepo *repository.WorkScheduleRepository) *AttendanceHandler {
	return &AttendanceHandler{
		repo: repo,
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

func (h *AttendanceHandler) ScanQRCode(c *fiber.Ctx) error {
	var payload models.QRCodeScanPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payload tidak valid: " + err.Error()})
	}

	// 1-4. Validasi QR Code dan User (kode Anda yang sudah ada tetap di sini)
	// ... (kode validasi QR code dan User ID) ...
    userID, _ := primitive.ObjectIDFromHex(payload.UserID) // Asumsi sudah valid
	today := time.Now().Format("2006-01-02")
    
	// 5. Cek duplikasi check-in (kode Anda yang sudah ada)
	// ...

	// ==================================================================
	// ✨ LANGKAH BARU: LOGIKA PENYESUAIAN DENGAN JADWAL KERJA ✨
	// ==================================================================

	// 6. Ambil jadwal kerja untuk hari ini.
	// Karena jadwal sekarang umum, kita hanya perlu mencari berdasarkan tanggal.
	schedule, err := h.workScheduleRepo.FindByDate(today)
	if err != nil || len(schedule) == 0 {
		// Jika tidak ada jadwal kerja hari ini (misalnya hari libur atau Minggu), tolak check-in.
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tidak ada jadwal kerja yang aktif untuk hari ini.",
		})
	}
	// Kita ambil jadwal pertama yang ditemukan untuk hari itu
	todaysSchedule := schedule[0]

	// 7. Tentukan status kehadiran (Tepat Waktu atau Terlambat)
	loc, _ := time.LoadLocation("Asia/Jakarta") // Zona waktu WIB
	currentTimeInWIB := time.Now().In(loc)
	
	// Parsing jam masuk dari jadwal. Format: "15:04"
	scheduledStartTime, err := time.ParseInLocation("15:04", todaysSchedule.StartTime, loc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses waktu jadwal."})
	}
	// Gabungkan dengan tanggal hari ini untuk perbandingan yang akurat
	scheduledCheckInTime := time.Date(
		currentTimeInWIB.Year(), currentTimeInWIB.Month(), currentTimeInWIB.Day(),
		scheduledStartTime.Hour(), scheduledStartTime.Minute(), 0, 0, loc,
	)

	// Berikan toleransi keterlambatan (misalnya, 15 menit)
	gracePeriod := 15 * time.Minute
	latestCheckInTime := scheduledCheckInTime.Add(gracePeriod)

	var attendanceStatus string
	if currentTimeInWIB.After(latestCheckInTime) {
		attendanceStatus = "Terlambat"
	} else {
		attendanceStatus = "Tepat Waktu"
	}

	// ==================================================================
	// ✨ AKHIR DARI LOGIKA BARU ✨
	// ==================================================================

	// 8. Proses CHECK-IN dengan status yang sudah ditentukan
	checkInTimeFormatted := currentTimeInWIB.Format("15:04") // Format 24 jam untuk konsistensi

	newAttendance := models.Attendance{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Date:      today,
		CheckIn:   checkInTimeFormatted,
		CheckOut:  "",
		Status:    attendanceStatus, // <-- GUNAKAN STATUS YANG SUDAH DITENTUKAN
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = h.repo.CreateAttendance(c.Context(), &newAttendance)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal melakukan check-in: " + err.Error()})
	}

	// 9. Berikan respons sukses check-in
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": fmt.Sprintf("Berhasil check-in pukul %s. Status: %s", newAttendance.CheckIn, newAttendance.Status),
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
	const QR_CODE_DURATION = 30 * time.Second // Durasi QR Code aktif (misal: 30 detik)

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	todayStr := time.Now().Format("2006-01-02")
	currentTime := time.Now()

	// Cari QR Code yang masih aktif untuk hari ini
	existingQRCode, err := h.repo.FindActiveQRCodeByDate(ctx, todayStr)

	// Jika ditemukan QR code yang aktif dan belum kadaluarsa, kembalikan itu
	if err == nil && existingQRCode != nil && currentTime.Before(existingQRCode.ExpiresAt) {
		// ... (kode untuk mengembalikan existingQRCode) ...
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

	// Jika tidak ada QR Code aktif atau sudah kadaluarsa, buat yang baru
	uniqueCode := uuid.New().String()
	// --- Pastikan expiresAt DITAMBAHKAN dari currentTime ---
	expiresAt := currentTime.Add(QR_CODE_DURATION) // <-- INI YANG HARUS DIPASTIKAN BENAR

	newQRCode := &models.QRCode{
		ID:        primitive.NewObjectID(),
		Code:      uniqueCode,
		Date:      todayStr,
		ExpiresAt: expiresAt, // <-- Pastikan menggunakan expiresAt yang baru dihitung
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}

	// ... (kode penyimpanan dan pengembalian respons) ...
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

	// Gunakan UserID dari claims untuk keamanan
	userID := claims.UserID

	// --- FindAttendanceByUserID sekarang menerima context ---
	attendanceHistory, err := h.repo.FindAttendanceByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil riwayat kehadiran: " + err.Error(),
		})
	}

	// Jika tidak ada data, kembalikan array kosong, bukan error
	if attendanceHistory == nil {
		return c.Status(fiber.StatusOK).JSON([]models.Attendance{})
	}

	return c.Status(fiber.StatusOK).JSON(attendanceHistory)
}
