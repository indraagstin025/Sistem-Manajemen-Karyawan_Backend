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
// @Description Membuat pengajuan cuti atau sakit baru.
// @Tags Leave Request
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param request_type formData string true "Jenis Pengajuan (Cuti, Sakit)" Enums(Cuti, Sakit)
// @Param start_date formData string true "Tanggal Mulai (YYYY-MM-DD)"
// @Param end_date formData string true "Tanggal Selesai (YYYY-MM-DD)"
// @Param reason formData string true "Alasan Pengajuan"
// @Param attachment formData file false "Lampiran (Wajib untuk Sakit, maks 2MB)"
// @Success 201 {object} object{message=string, request=models.LeaveRequest} "Pengajuan berhasil dikirim"
// @Failure 400 {object} object{error=string} "Input tidak valid"
// @Failure 403 {object} object{error=string} "Akses ditolak (misal: sudah mengajukan cuti bulan ini)"
// @Failure 500 {object} object{error=string} "Kesalahan server internal"
// @Router /leave-requests [post]
func (h *LeaveRequestHandler) CreateLeaveRequest(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Klaim token tidak valid atau sesi rusak"})
	}

	requestType := c.FormValue("request_type")
	startDateStr := c.FormValue("start_date")
	endDateStr := c.FormValue("end_date")
	reason := c.FormValue("reason")

	if requestType == "" || startDateStr == "" || endDateStr == "" || reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Jenis pengajuan, tanggal mulai, tanggal selesai, dan alasan wajib diisi."})
	}

	if requestType != "Cuti" && requestType != "Sakit" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Jenis pengajuan tidak valid. Hanya 'Cuti' atau 'Sakit' yang diterima."})
	}

	parsedStartDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format tanggal mulai tidak valid. Gunakan YYYY-MM-DD."})
	}

	// Hanya terapkan batasan ini untuk jenis pengajuan "Cuti"
	if requestType == "Cuti" {
		currentMonth := parsedStartDate.Month()
		currentYear := parsedStartDate.Year()

		// --- LOGIKA BARU: BATASAN 12 KALI PER TAHUN ---
		// PENTING: Pastikan ini menghitung jumlah dokumen (pengajuan) bukan durasi hari.
		annualLeaveCount, err := h.leaveRepo.CountByUserIDYearAndType(c.Context(), claims.UserID, currentYear, requestType)
		if err != nil {
			log.Printf("ERROR: Gagal memeriksa jumlah pengajuan Cuti tahunan untuk user %s di tahun %d: %v", claims.UserID.Hex(), currentYear, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memeriksa batasan pengajuan Cuti tahunan."})
		}
		if annualLeaveCount >= 12 { // Jika sudah 12 kali pengajuan (bukan hari)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": fmt.Sprintf("Anda telah mencapai batas maksimal 12 kali pengajuan 'Cuti' untuk tahun %d ini.", currentYear),
			})
		}
		// --- AKHIR BATASAN TAHUNAN ---

		// --- LOGIKA YANG SUDAH ADA: BATASAN 1 KALI PER BULAN ---
		existingCount, err := h.leaveRepo.CountByUserIDMonthAndType(c.Context(), claims.UserID, currentYear, currentMonth, requestType)
		if err != nil {
			log.Printf("ERROR: Gagal memeriksa jumlah pengajuan Cuti untuk user %s di bulan %s %d: %v", claims.UserID.Hex(), currentMonth, currentYear, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memeriksa batasan pengajuan Cuti."})
		}
		if existingCount > 0 { // Jika sudah ada 1 pengajuan cuti di bulan ini
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": fmt.Sprintf("Anda hanya dapat mengajukan 'Cuti' satu kali dalam bulan %s %d.", currentMonth.String(), currentYear),
			})
		}
	}

	// Logika pengecekan duplikasi untuk Sakit (bisa dipertahankan jika relevan)
	if requestType == "Sakit" {
		startDate, _ := time.Parse("2006-01-02", startDateStr)
		endDate, _ := time.Parse("2006-01-02", endDateStr)

		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			existing, err := h.leaveRepo.FindByUserAndDateAndType(c.Context(), claims.UserID, dateStr, requestType)
			if err != nil && err != mongo.ErrNoDocuments {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Gagal memeriksa pengajuan sebelumnya",
				})
			}
			if existing != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Anda sudah mengajukan %s untuk tanggal %s.", requestType, dateStr),
				})
			}
		}
	}

	var attachmentURL string
	file, err := c.FormFile("attachment")
	if err == nil && file != nil {
		if file.Size > 2*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ukuran file maksimal 2MB"})
		}
		allowedExtensions := map[string]bool{".pdf": true, ".jpg": true, ".jpeg": true, ".png": true}
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !allowedExtensions[ext] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format file tidak didukung (hanya .pdf, .jpg, .jpeg, .png)"})
		}

		bucket, err := config.GetGridFSBucket()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengakses penyimpanan file"})
		}
		src, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuka file"})
		}
		defer src.Close()

		uploadFileName := fmt.Sprintf("%d_%s", time.Now().Unix(), strings.ReplaceAll(file.Filename, " ", "_"))
		uploadStream, err := bucket.OpenUploadStream(uploadFileName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal upload file"})
		}
		defer uploadStream.Close()

		if _, err := io.Copy(uploadStream, src); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file"})
		}
		attachmentURL = fmt.Sprintf("/api/v1/files/%s", uploadStream.FileID.(primitive.ObjectID).Hex())
	} else if requestType == "Sakit" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Lampiran wajib untuk pengajuan Sakit"})
	}

	newRequest := &models.LeaveRequest{
		ID:            primitive.NewObjectID(),
		UserID:        claims.UserID,
		StartDate:     startDateStr,
		EndDate:       endDateStr,
		Reason:        reason,
		Status:        "pending", // Status awal selalu 'pending'
		RequestType:   requestType,
		AttachmentURL: attachmentURL,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, createErr := h.leaveRepo.Create(newRequest)
	if createErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan data pengajuan"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Pengajuan berhasil dikirim", "request": newRequest})
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

	requests, err := h.leaveRepo.FindAll()
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

	requests, err := h.leaveRepo.FindByUserID(c.Context(), claims.UserID)
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
	fileURL := fmt.Sprintf("/api/v1/files/%s", fileID.Hex())

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

	// Validasi status hanya boleh "approved" atau "rejected"
	validStatuses := map[string]bool{"approved": true, "rejected": true}
	if !validStatuses[payload.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Status tidak valid. Hanya 'approved' atau 'rejected' yang diperbolehkan.",
		})
	}

	if len(payload.Note) > 500 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Catatan admin terlalu panjang (maksimal 500 karakter)",
		})
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

	// ✅ Validasi agar hanya Cuti dan Sakit yang boleh
	if originalRequest.RequestType != "Cuti" && originalRequest.RequestType != "Sakit" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Jenis pengajuan tidak valid (%s). Hanya 'Cuti' dan 'Sakit' yang diperbolehkan.", originalRequest.RequestType),
		})
	}

	// Proses update status di database
	updateResult, err := h.leaveRepo.UpdateStatus(reqID, payload.Status, payload.Note)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui status pengajuan cuti",
		})
	}
	if updateResult.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pengajuan dengan ID ini tidak ditemukan"})
	}

	// Parse tanggal
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

	// Proses absensi berdasarkan status baru
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
				if existingAttendance.Status == "Sakit" || existingAttendance.Status == "Cuti" {
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

// GetLeaveSummary godoc
// @Summary Get Leave Request Summary for current user
// @Description Mengambil ringkasan jumlah pengajuan cuti (per bulan dan per tahun) untuk karyawan yang sedang login.
// @Tags Leave Request
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.LeaveSummaryResponse "Ringkasan pengajuan cuti berhasil diambil"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 500 {object} object{error=string} "Gagal mengambil ringkasan pengajuan"
// @Router /leave-requests/summary [get]
func (h *LeaveRequestHandler) GetLeaveSummary(c *fiber.Ctx) error {
	// Pastikan user sudah login dan klaim token valid
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Klaim token tidak valid atau sesi rusak"})
	}

	// Dapatkan tahun dan bulan saat ini
	currentYear := time.Now().Year()
	currentMonth := time.Now().Month()

	// Hitung pengajuan cuti untuk bulan ini
	// Panggil fungsi CountByUserIDMonthAndType dari repository Anda
	// Hanya hitung pengajuan dengan RequestType "Cuti"
	monthlyCount, err := h.leaveRepo.CountByUserIDMonthAndType(c.Context(), claims.UserID, currentYear, currentMonth, "Cuti")
	if err != nil {
		log.Printf("ERROR: Gagal menghitung cuti bulanan untuk user %s: %v", claims.UserID.Hex(), err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil ringkasan cuti bulanan."})
	}

	// Hitung pengajuan cuti untuk tahun ini
	// Panggil fungsi CountByUserIDYearAndType dari repository Anda
	// Hanya hitung pengajuan dengan RequestType "Cuti"
	annualCount, err := h.leaveRepo.CountByUserIDYearAndType(c.Context(), claims.UserID, currentYear, "Cuti")
	if err != nil {
		log.Printf("ERROR: Gagal menghitung cuti tahunan untuk user %s: %v", claims.UserID.Hex(), err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil ringkasan cuti tahunan."})
	}

	// Buat respons menggunakan struct LeaveSummaryResponse
	response := models.LeaveSummaryResponse{
		CurrentMonthLeaveCount: monthlyCount,
		AnnualLeaveCount:       annualCount,
	}

	// Kirim respons JSON
	return c.Status(fiber.StatusOK).JSON(response)
}