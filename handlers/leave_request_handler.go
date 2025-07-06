package handlers

import (
	"fmt"
	"time"

	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/pkg/paseto"
	"Sistem-Manajemen-Karyawan/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeaveRequestHandler struct {
	leaveRepo      repository.LeaveRequestRepository
	attendanceRepo repository.AttendanceRepository
}

func NewLeaveRequestHandler(leaveRepo repository.LeaveRequestRepository, attendanceRepo repository.AttendanceRepository) *LeaveRequestHandler {
	return &LeaveRequestHandler{
		leaveRepo:      leaveRepo,
		attendanceRepo: attendanceRepo,
	}
}

func (h *LeaveRequestHandler) CreateLeaveRequest(c *fiber.Ctx) error {
	var payload models.LeaveRequestCreatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payload tidak valid"})
	}

	claims, ok := c.Locals("user").(*paseto.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Klaim token tidak valid"})
	}

	newRequest := &models.LeaveRequest{
		ID:        primitive.NewObjectID(),
		UserID:    claims.UserID,
		StartDate: payload.StartDate,
		EndDate:   payload.EndDate,
		Reason:    payload.Reason,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := h.leaveRepo.Create(newRequest)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat pengajuan"})
	}

	return c.Status(fiber.StatusCreated).JSON(newRequest)
}

func (h *LeaveRequestHandler) GetAllLeaveRequests(c *fiber.Ctx) error {
	requests, err := h.leaveRepo.FindAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data pengajuan"})
	}
	return c.Status(fiber.StatusOK).JSON(requests)
}

func (h *LeaveRequestHandler) UploadAttachment(c *fiber.Ctx) error {
	id := c.Params("id")
	reqID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID pengajuan tidak valid"})
	}

	file, err := c.FormFile("attachment")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File tidak ditemukan"})
	}

	uniqueFileName := fmt.Sprintf("%d-%s", time.Now().Unix(), file.Filename)
	filePath := fmt.Sprintf("./uploads/attachments/%s", uniqueFileName)

	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file"})
	}

	fileURL := fmt.Sprintf("/uploads/attachments/%s", uniqueFileName)

	_, err = h.leaveRepo.UpdateAttachmentURL(reqID, fileURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan URL file ke database"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "File berhasil diunggah",
		"file_url": fileURL,
	})
}

func (h *LeaveRequestHandler) UpdateLeaveRequestStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	reqID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID tidak valid"})
	}

	var payload models.LeaveRequestUpdatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payload tidak valid"})
	}

	updateResult, err := h.leaveRepo.UpdateStatus(reqID, payload.Status, payload.Note)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui status"})
	}

	if updateResult.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pengajuan dengan ID ini tidak ditemukan"})
	}

	if payload.Status == "approved" {
		request, err := h.leaveRepo.FindByID(reqID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menemukan data pengajuan setelah diupdate"})
		}

		startDate, _ := time.Parse("2006-01-02", request.StartDate)
		endDate, _ := time.Parse("2006-01-02", request.EndDate)

		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
attendanceRecord := &models.Attendance{
    ID:        primitive.NewObjectID(),
    UserID:    request.UserID,
    Date:      d.Format("2006-01-02"),
    Status:    request.RequestType, // Diubah dari "Izin" menjadi dinamis
    Note:      "Disetujui: " + request.Reason,
    CreatedAt: time.Now(),
    UpdatedAt: time.Now(),
}
			h.attendanceRepo.CreateAttendance(attendanceRecord)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Status pengajuan berhasil diperbarui"})
}
