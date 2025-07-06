package handlers

import (
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/pkg/paseto"
	"Sistem-Manajemen-Karyawan/repository"
)

type AttendanceHandler struct {
	repo repository.AttendanceRepository
}

func NewAttendanceHandler(repo repository.AttendanceRepository) *AttendanceHandler {
	return &AttendanceHandler{repo: repo}
}

func (h *AttendanceHandler) ScanQRCode(c *fiber.Ctx) error {
	var payload models.QRCodeScanPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payload tidak valid: " + err.Error()})
	}

	qrCode, err := h.repo.FindQRCodeByValue(payload.QRCodeValue)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "QR Code tidak ditemukan atau tidak valid."})
	}

	if time.Now().After(qrCode.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "QR Code sudah kadaluarsa."})
	}

	today := time.Now().Format("2006-01-02")
	if qrCode.Date != today {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "QR Code ini tidak berlaku untuk hari ini."})
	}

	userID, _ := primitive.ObjectIDFromHex(payload.UserID)
	for _, usedID := range qrCode.UsedBy {
		if usedID == userID {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Anda sudah menggunakan QR Code ini untuk absensi."})
		}
	}

	attendance, err := h.repo.FindAttendanceByUserAndDate(userID, today)
	if err == nil {
		if attendance.CheckOut == "" {
			currentTime := time.Now().Format("15:04")
			_, err := h.repo.UpdateAttendanceCheckout(attendance.ID, currentTime)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal melakukan check-out."})
			}
			h.repo.MarkQRCodeAsUsed(qrCode.ID, userID)
			return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Berhasil check-out pukul " + currentTime})
		} else {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Anda sudah melakukan check-in dan check-out hari ini."})
		}
	}

	newAttendance := models.Attendance{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Date:      today,
		CheckIn:   time.Now().Format("15:04"),
		CheckOut:  "",
		Status:    "Hadir",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = h.repo.CreateAttendance(&newAttendance)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal melakukan check-in."})
	}

	h.repo.MarkQRCodeAsUsed(qrCode.ID, userID)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Berhasil check-in pukul " + newAttendance.CheckIn})
}

func (h *AttendanceHandler) GenerateQRCode(c *fiber.Ctx) error {
	uniqueCode := uuid.New().String()
	today := time.Now()
	expiresAt := time.Date(today.Year(), today.Month(), today.Day(), 23, 0, 0, 0, today.Location())

	newQRCode := &models.QRCode{
		ID:        primitive.NewObjectID(),
		Code:      uniqueCode,
		Date:      today.Format("2006-01-02"),
		ExpiresAt: expiresAt,
		UsedBy:    []primitive.ObjectID{},
		CreatedAt: today,
	}

	_, err := h.repo.CreateQRCode(newQRCode)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan data QR Code."})
	}

	png, err := qrcode.Encode(uniqueCode, qrcode.Medium, 256)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat gambar QR Code."})
	}

	encodedString := base64.StdEncoding.EncodeToString(png)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":       "QR Code berhasil dibuat",
		"qr_code_image": "data:image/png;base64," + encodedString,
		"expires_at":    expiresAt,
	})
}

func (h *AttendanceHandler) GetMyAttendanceHistory(c *fiber.Ctx) error {

	claims, ok := c.Locals("user").(*paseto.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Tidak terautentikasi atau klaim token tidak valid",
		})
	}

	userID := claims.UserID

	attendanceHistory, err := h.repo.FindAttendanceByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil riwayat kehadiran",
		})
	}

	if attendanceHistory == nil {
		return c.Status(fiber.StatusOK).JSON([]models.Attendance{})
	}

	return c.Status(fiber.StatusOK).JSON(attendanceHistory)
}

func (h *AttendanceHandler) GetTodayAttendance(c *fiber.Ctx) error {
	attendanceList, err := h.repo.GetTodayAttendanceWithUserDetails()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil daftar kehadiran",
		})
	}

	if attendanceList == nil {

		return c.Status(fiber.StatusOK).JSON([]models.AttendanceWithUser{})
	}

	return c.Status(fiber.StatusOK).JSON(attendanceList)
}
