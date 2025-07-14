package handlers

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

// CreateLeaveRequest godoc
// @Summary Create Leave Request
// @Description Membuat pengajuan izin/cuti/sakit baru
// @Tags Leave Request
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body models.LeaveRequestCreatePayload true "Data pengajuan izin"
// @Success 201 {object} models.LeaveRequest "Pengajuan berhasil dibuat"
// @Failure 400 {object} object{error=string} "Payload tidak valid"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 500 {object} object{error=string} "Gagal membuat pengajuan"
// @Router /leave-requests [post]
func (h *LeaveRequestHandler) CreateLeaveRequest(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Klaim token tidak valid atau sesi rusak"})
	}

	// Ambil data form
	requestType := c.FormValue("request_type")
	startDate := c.FormValue("start_date")
	endDate := c.FormValue("end_date")
	reason := c.FormValue("reason")

	if requestType == "" || startDate == "" || endDate == "" || reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Jenis pengajuan, tanggal mulai, tanggal selesai, dan alasan wajib diisi.",
		})
	}

	var attachmentURL string
	file, _ := c.FormFile("attachment")

	if file != nil {
		// Validasi ukuran maksimal 2MB
		if file.Size > 2*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ukuran file maksimal 2MB"})
		}

		// Validasi ekstensi file
		allowedExtensions := map[string]bool{
			".pdf":  true,
			".doc":  true,
			".docx": true,
			".jpg":  true,
			".jpeg": true,
			".png":  true,
		}
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !allowedExtensions[ext] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format file tidak didukung"})
		}

		// Validasi khusus untuk Sakit (harus PDF)
		if requestType == "Sakit" && ext != ".pdf" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Jenis pengajuan Sakit harus berupa file PDF"})
		}

		// Upload file ke GridFS
		bucket, err := config.GetGridFSBucket()
		if err != nil {
			log.Printf("ERROR: Gagal mendapatkan bucket GridFS: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengakses penyimpanan file"})
		}

		src, err := file.Open()
		if err != nil {
			log.Printf("ERROR: Gagal membuka file: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuka file"})
		}
		defer src.Close()

		uploadFileName := fmt.Sprintf("%d_%s", time.Now().Unix(), strings.ReplaceAll(file.Filename, " ", "_"))
		uploadStream, err := bucket.OpenUploadStream(uploadFileName)
		if err != nil {
			log.Printf("ERROR: Gagal membuka stream GridFS: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal upload file"})
		}
		defer uploadStream.Close()

		if _, err := io.Copy(uploadStream, src); err != nil {
			log.Printf("ERROR: Gagal menyalin ke GridFS: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file"})
		}

		attachmentURL = fmt.Sprintf("/api/v1/files/%s", uploadStream.FileID.(primitive.ObjectID).Hex())
	} else if requestType == "Sakit" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Lampiran wajib untuk pengajuan Sakit"})
	}

	// Simpan data pengajuan ke database
	newRequest := &models.LeaveRequest{
		ID:            primitive.NewObjectID(),
		UserID:        claims.UserID,
		StartDate:     startDate,
		EndDate:       endDate,
		Reason:        reason,
		Status:        "pending",
		RequestType:   requestType,
		AttachmentURL: attachmentURL,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, createErr := h.leaveRepo.Create(newRequest)
	if createErr != nil {
		log.Printf("ERROR: Gagal menyimpan leave request ke DB: %v", createErr)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan data pengajuan"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Pengajuan berhasil dikirim",
		"request": newRequest,
	})
}


// GetAllLeaveRequests godoc
// @Summary Get All Leave Requests
// @Description Mengambil semua pengajuan izin/cuti/sakit (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// PERBAIKAN DI SINI: Ubah tipe di Swagger doc
// @Success 200 {array} models.LeaveRequestWithUser "Daftar pengajuan berhasil diambil dengan detail user"
// @Failure 500 {object} object{error=string} "Gagal mengambil data pengajuan"
// @Router /leave-requests [get]
func (h *LeaveRequestHandler) GetAllLeaveRequests(c *fiber.Ctx) error {
	// PERBAIKAN DI SINI: Deklarasikan variabel dengan tipe yang benar
	requests, err := h.leaveRepo.FindAll() // Sekarang FindAll mengembalikan []models.LeaveRequestWithUser
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal mengambil data pengajuan: %v", err)})
	}
	return c.Status(fiber.StatusOK).JSON(requests)
}

// GetMyLeaveRequests godoc
// @Summary Get Leave Requests for current user
// @Description Mengambil semua pengajuan cuti/izin/sakit untuk karyawan yang sedang login
// @Tags Leave Request
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.LeaveRequest "Daftar pengajuan berhasil diambil"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 500 {object} object{error=string} "Gagal mengambil data pengajuan"
// @Router /leave-requests/my-requests [get]
func (h *LeaveRequestHandler) GetMyLeaveRequests(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Klaim token tidak valid atau sesi rusak"})
	}

	// Memanggil repository untuk mencari pengajuan berdasarkan UserID
	requests, err := h.leaveRepo.FindByUserID(c.Context(), claims.UserID) // <-- Membutuhkan FindByUserID di repository
	if err != nil {
		log.Printf("ERROR: Gagal mengambil pengajuan cuti untuk user %s: %v", claims.UserID.Hex(), err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal mengambil data pengajuan: %v", err)})
	}

	return c.Status(fiber.StatusOK).JSON(requests)
}

// UploadAttachment godoc
// @Summary Upload Attachment for Leave Request
// @Description Mengunggah file lampiran untuk pengajuan izin/cuti/sakit
// @Tags Leave Request
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "Leave Request ID"
// @Param attachment formData file true "File lampiran"
// @Success 200 {object} object{message=string,file_url=string} "File berhasil diunggah"
// @Failure 400 {object} object{error=string} "ID tidak valid atau file tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal menyimpan file"
// @Router /leave-requests/{id}/attachment [post]
func (h *LeaveRequestHandler) UploadAttachment(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Println("[UploadAttachment] ID:", id)

	reqID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID pengajuan tidak valid"})
	}

	// Ambil file dari form-data
	fileHeader, err := c.FormFile("attachment")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File tidak ditemukan"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuka file"})
	}
	defer file.Close()

	bucket, err := config.GetGridFSBucket()
	if err != nil {
		log.Println("Gagal membuat bucket GridFS:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal inisialisasi penyimpanan file"})
	}

	uploadStream, err := bucket.OpenUploadStream(fileHeader.Filename)
	if err != nil {
		log.Println("Gagal membuka upload stream GridFS:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal mengunggah file"})
	}
	defer uploadStream.Close()

	fileSize, err := io.Copy(uploadStream, file)
	if err != nil {
		log.Println("Gagal menulis file ke GridFS:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan file"})
	}

	log.Printf("Berhasil upload ke GridFS (%d bytes)\n", fileSize)

	fileID := uploadStream.FileID.(primitive.ObjectID)
	fileURL := fmt.Sprintf("/api/v1/files/%s", fileID.Hex()) // disimpan di database

	// Simpan fileURL ke database leave request
	_, err = h.leaveRepo.UpdateAttachmentURL(reqID, fileURL)
	if err != nil {
		log.Println("Gagal menyimpan URL lampiran ke DB:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan file ke database"})
	}

	return c.Status(200).JSON(fiber.Map{
		"message":  "File berhasil diunggah",
		"file_url": fileURL,
	})
}


// UpdateLeaveRequestStatus godoc
// @Summary Update Leave Request Status
// @Description Memperbarui status pengajuan izin/cuti/sakit (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Leave Request ID"
// @Param payload body models.LeaveRequestUpdatePayload true "Data update status"
// @Success 200 {object} object{message=string} "Status pengajuan berhasil diperbarui"
// @Failure 400 {object} object{error=string} "ID tidak valid atau payload tidak valid"
// @Failure 404 {object} object{error=string} "Pengajuan tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal memperbarui status"
// @Router /leave-requests/{id}/status [put]
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

	originalRequest, err := h.leaveRepo.FindByID(reqID)
	if err != nil {
		if err == mongo.ErrNoDocuments || err.Error() == "gagal menemukan pengajuan berdasarkan ID" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pengajuan tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Gagal mencari pengajuan cuti: %v", err),
		})
	}
	if originalRequest == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pengajuan tidak ditemukan"})
	}

	updateResult, err := h.leaveRepo.UpdateStatus(reqID, payload.Status, payload.Note)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui status pengajuan cuti",
		})
	}
	if updateResult.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pengajuan dengan ID ini tidak ditemukan"})
	}

	startDate, parseErr := time.Parse("2006-01-02", originalRequest.StartDate)
	if parseErr != nil {
		log.Printf("ERROR: Gagal parse start_date %s: %v", originalRequest.StartDate, parseErr)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Format tanggal mulai tidak valid.",
		})
	}
	endDate, parseErr := time.Parse("2006-01-02", originalRequest.EndDate)
	if parseErr != nil {
		log.Printf("ERROR: Gagal parse end_date %s: %v", originalRequest.EndDate, parseErr)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Format tanggal berakhir tidak valid.",
		})
	}

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		currentDateStr := d.Format("2006-01-02")

		existingAttendance, err := h.attendanceRepo.FindAttendanceByUserAndDate(
			c.Context(), originalRequest.UserID, currentDateStr,
		)
		if err != nil && err != mongo.ErrNoDocuments {
			log.Printf("ERROR: Gagal mencari absensi untuk user %s tanggal %s: %v",
				originalRequest.UserID.Hex(), currentDateStr, err)
			continue
		}

		if payload.Status == "approved" {
			if existingAttendance == nil {
				attendanceRecord := &models.Attendance{
					ID:        primitive.NewObjectID(),
					UserID:    originalRequest.UserID,
					Date:      currentDateStr,
					CheckIn:   "",
					CheckOut:  "",
					Status:    originalRequest.RequestType,
					Note:      fmt.Sprintf("Disetujui: %s. Catatan admin: %s", originalRequest.Reason, payload.Note),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				_, createErr := h.attendanceRepo.CreateAttendance(c.Context(), attendanceRecord)
				if createErr != nil {
					log.Printf("ERROR: Gagal menyimpan absensi baru untuk tanggal %s (approved): %v", currentDateStr, createErr)
				}
			} else {
				updatePayload := models.AttendanceUpdatePayload{
					Status:   originalRequest.RequestType,
					Note:     fmt.Sprintf("Disetujui: %s. Catatan admin: %s", originalRequest.Reason, payload.Note),
					CheckIn:  "",
					CheckOut: "",
				}
				_, updateErr := h.attendanceRepo.UpdateAttendance(c.Context(), existingAttendance.ID, &updatePayload)
				if updateErr != nil {
					log.Printf("ERROR: Gagal memperbarui absensi existing untuk tanggal %s (approved): %v", currentDateStr, updateErr)
				}
			}
		} else if payload.Status == "rejected" {
			if existingAttendance != nil {
				if existingAttendance.Status == "Sakit" || existingAttendance.Status == "Cuti" || existingAttendance.Status == "Izin" {
					if existingAttendance.CheckIn == "" && existingAttendance.CheckOut == "" {
						updatePayload := models.AttendanceUpdatePayload{
							Status:   "Tidak Absen",
							Note:     fmt.Sprintf("Pengajuan ditolak: %s. Catatan admin: %s", originalRequest.Reason, payload.Note),
							CheckIn:  "",
							CheckOut: "",
						}
						_, updateErr := h.attendanceRepo.UpdateAttendance(c.Context(), existingAttendance.ID, &updatePayload)
						if updateErr != nil {
							log.Printf("ERROR: Gagal memperbarui absensi existing untuk tanggal %s (rejected): %v", currentDateStr, updateErr)
						}
					}
				}
			} else {
				newAttendance := &models.Attendance{
					ID:        primitive.NewObjectID(),
					UserID:    originalRequest.UserID,
					Date:      currentDateStr,
					CheckIn:   "",
					CheckOut:  "",
					Status:    "Tidak Absen",
					Note:      fmt.Sprintf("Pengajuan ditolak: %s. Catatan admin: %s", originalRequest.Reason, payload.Note),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				_, createErr := h.attendanceRepo.CreateAttendance(c.Context(), newAttendance)
				if createErr != nil {
					log.Printf("ERROR: Gagal membuat absensi default untuk tanggal %s (rejected): %v", currentDateStr, createErr)
				}
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Status pengajuan berhasil diperbarui",
	})
}

