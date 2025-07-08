package handlers

import (
	"context" // Pastikan ini diimpor
	"encoding/base64"
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
}

func NewAttendanceHandler(repo repository.AttendanceRepository) *AttendanceHandler {
    return &AttendanceHandler{repo: repo}
}


func (h *AttendanceHandler) ScanQRCode(c *fiber.Ctx) error {
    var payload models.QRCodeScanPayload
    if err := c.BodyParser(&payload); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Payload tidak valid: " + err.Error(),
        })
    }

    qrCode, err := h.repo.FindQRCodeByValue(c.Context(), payload.QRCodeValue)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Terjadi kesalahan saat mencari QR Code: " + err.Error(),
        })
    }
    if qrCode == nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "QR Code tidak ditemukan atau tidak valid.",
        })
    }

    // Validasi kadaluarsa dan tanggal QR
    if time.Now().After(qrCode.ExpiresAt) {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "QR Code sudah kadaluarsa.",
        })
    }

    today := time.Now().Format("2006-01-02")
    if qrCode.Date != today {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "QR Code ini tidak berlaku untuk hari ini.",
        })
    }

    // Validasi ID user
    userID, err := primitive.ObjectIDFromHex(payload.UserID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "User ID tidak valid.",
        })
    }

    // Cek apakah user sudah absen hari ini
    attendance, err := h.repo.FindAttendanceByUserAndDate(c.Context(), userID, today)
    if err == nil && attendance != nil {
        if attendance.CheckOut == "" {
            // ✅ Proses CHECK-OUT
            currentTime := time.Now().Format("15:04")
            _, err := h.repo.UpdateAttendanceCheckout(c.Context(), attendance.ID, currentTime)
            if err != nil {
                return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                    "error": "Gagal melakukan check-out: " + err.Error(),
                })
            }
            return c.Status(fiber.StatusOK).JSON(fiber.Map{
                "message": "Berhasil check-out pukul " + currentTime,
            })
        } else {
            // ❌ Sudah check-in dan check-out
            return c.Status(fiber.StatusConflict).JSON(fiber.Map{
                "error": "Anda sudah melakukan check-in dan check-out hari ini.",
            })
        }
    }

    // ✅ Proses CHECK-IN (belum ada absensi hari ini)
    newAttendance := models.Attendance{
        ID:        primitive.NewObjectID(),
        UserID:    userID,
        Date:      today,
        CheckIn:   time.Now().Format("15:04"),
        CheckOut:  "",
        Status:    "Hadir", // default hadir
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    _, err = h.repo.CreateAttendance(c.Context(), &newAttendance)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Gagal melakukan check-in: " + err.Error(),
        })
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "message": "Berhasil check-in pukul " + newAttendance.CheckIn,
    })
}




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
// @Description Mengambil daftar kehadiran karyawan untuk hari ini dengan detail user.
// @Tags Attendance
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
// @Description Mengambil seluruh riwayat absensi untuk user yang sedang login.
// @Tags Attendance
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