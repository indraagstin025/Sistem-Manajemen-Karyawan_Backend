package handlers

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"Sistem-Manajemen-Karyawan/models"
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

	// Mengambil data dari form-data (bukan JSON body)
	requestType := c.FormValue("request_type")
	startDate := c.FormValue("start_date")
	endDate := c.FormValue("end_date")
	reason := c.FormValue("reason")

	// Validasi dasar field wajib (non-file)
	if requestType == "" || startDate == "" || endDate == "" || reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Jenis pengajuan, tanggal mulai, tanggal selesai, dan alasan wajib diisi."})
	}

	var attachmentURL string
	file, err := c.FormFile("attachment")

	// Logika penanganan dan validasi file lampiran
	if file != nil { // Jika ada file yang diunggah
		// Validasi ukuran file (maks 2MB, sesuai frontend)
		if file.Size > 2*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ukuran file lampiran terlalu besar. Maksimal 2MB."})
		}

		// Validasi tipe file
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format file lampiran tidak didukung. Hanya PDF, DOC/DOCX, JPG, PNG."})
		}

		// Validasi khusus untuk jenis 'Sakit': harus PDF
		if requestType == "Sakit" && ext != ".pdf" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Untuk pengajuan Sakit, lampiran harus berupa file PDF."})
		}

		// Simpan file ke disk
		uniqueFileName := fmt.Sprintf("%d-%s%s", time.Now().Unix(), strings.ReplaceAll(file.Filename[:len(file.Filename)-len(ext)], " ", "_"), ext)
		filePath := fmt.Sprintf("./uploads/attachments/%s", uniqueFileName)
		attachmentURL = fmt.Sprintf("/uploads/attachments/%s", uniqueFileName)

		if err := c.SaveFile(file, filePath); err != nil {
			log.Printf("ERROR: Gagal menyimpan file lampiran ke disk: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal menyimpan file lampiran: %v", err)})
		}
	} else if requestType == "Sakit" { // Jika jenisnya Sakit TAPI tidak ada file
		// Ini adalah kondisi di mana file attachment WAJIB untuk 'Sakit'
		// Error ini akan terpicu jika file tidak ada sama sekali atau ada masalah saat membaca form file
		if err != nil && strings.Contains(err.Error(), "no such file") { // Pastikan errornya memang karena tidak ada file
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Untuk pengajuan Sakit, lampiran (surat dokter) wajib diunggah."})
		} else if err != nil { // Error lain saat memproses file
			log.Printf("ERROR: Terjadi error saat membaca file attachment: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal memproses lampiran: %v", err)})
		}
		// Jika file == nil dan err == nil (misal field attachment kosong tapi tidak wajib), maka ini ok
	}
	// Jika requestType BUKAN Sakit, maka file tidak wajib, jadi file == nil itu tidak masalah di sini.

	newRequest := &models.LeaveRequest{
		ID:            primitive.NewObjectID(),
		UserID:        claims.UserID,
		StartDate:     startDate,
		EndDate:       endDate,
		Reason:        reason,
		Status:        "pending", // Status awal selalu pending
		RequestType:   requestType,
		AttachmentURL: attachmentURL, // Simpan URL lampiran
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, createErr := h.leaveRepo.Create(newRequest)
	if createErr != nil {
		log.Printf("ERROR: Gagal membuat pengajuan cuti/izin di database: %v", createErr)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat pengajuan cuti/izin."})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Pengajuan berhasil dikirim dan menunggu persetujuan admin.",
		"request": newRequest, // Mengembalikan objek request yang baru dibuat
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
	reqID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID pengajuan tidak valid"})
	}

	file, err := c.FormFile("attachment")
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File tidak ditemukan"})
		}
		log.Printf("ERROR: Gagal mengambil file lampiran: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal mengambil file: %v", err)})
	}

	uniqueFileName := fmt.Sprintf("%d-%s", time.Now().Unix(), file.Filename)
	filePath := fmt.Sprintf("./uploads/attachments/%s", uniqueFileName)
	fileURL := fmt.Sprintf("/uploads/attachments/%s", uniqueFileName) // URL relatif untuk disimpan di DB

	if err := c.SaveFile(file, filePath); err != nil {
		log.Printf("ERROR: Gagal menyimpan file lampiran ke disk: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal menyimpan file: %v", err)})
	}

	_, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	_, err = h.leaveRepo.UpdateAttachmentURL(reqID, fileURL)
	if err != nil {
		log.Printf("ERROR: Gagal menyimpan URL file ke database untuk reqID %s: %v", reqID.Hex(), err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal menyimpan URL file ke database: %v", err)})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
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
			// Periksa jika errornya karena dokumen tidak ditemukan
			if err.Error() == "departemen tidak ditemukan" || err.Error() == "gagal menemukan pengajuan berdasarkan ID" { // Sesuaikan pesan error dari FindByID Anda
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pengajuan tidak ditemukan setelah update status."})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Gagal menemukan data pengajuan setelah diupdate: %v", err)})
		}
		if request == nil { // Tambahkan cek nil jika FindByID mengembalikan nil untuk tidak ditemukan
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pengajuan tidak ditemukan setelah diupdate."})
		}

		startDate, _ := time.Parse("2006-01-02", request.StartDate)
		endDate, _ := time.Parse("2006-01-02", request.EndDate)

		// Logika untuk mencatat/memperbarui absensi di koleksi attendances
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			currentDateStr := d.Format("2006-01-02")

			// Cek apakah sudah ada entri absensi untuk user dan tanggal ini
			existingAttendance, err := h.attendanceRepo.FindAttendanceByUserAndDate(c.Context(), request.UserID, currentDateStr)
			if err != nil {
				log.Printf("ERROR: Gagal mencari absensi existing untuk user %s tanggal %s: %v", request.UserID.Hex(), currentDateStr, err)
				// Lanjutkan ke tanggal berikutnya atau tangani error lebih lanjut
				continue
			}

			if existingAttendance == nil {
				// Jika belum ada, buat entri absensi baru
				attendanceRecord := &models.Attendance{
					ID:        primitive.NewObjectID(),
					UserID:    request.UserID,
					Date:      currentDateStr,
					CheckIn:   "", // Tidak ada check-in/check-out untuk cuti/sakit
					CheckOut:  "",
					Status:    request.RequestType, // Menggunakan RequestType (Sakit/Cuti/Izin)
					Note:      "Disetujui: " + request.Reason,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				_, err := h.attendanceRepo.CreateAttendance(c.Context(), attendanceRecord)
				if err != nil {
					log.Printf("ERROR: Gagal menyimpan absensi baru untuk tanggal %s: %v", currentDateStr, err)
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": fmt.Sprintf("Gagal menyimpan absensi tanggal %s: %v", currentDateStr, err),
					})
				}
			} else {
				// Jika sudah ada, PERBARUI entri yang sudah ada
				// Pastikan AttendanceUpdatePayload Anda mendukung update Status dan Note
				updatePayload := models.AttendanceUpdatePayload{
					Status: request.RequestType, // Update status menjadi Sakit/Cuti/Izin
					Note:   "Disetujui: " + request.Reason,
					// CheckIn/CheckOut tidak diubah agar tidak menimpa jika sudah ada record hadir
					// Atau Anda bisa mengosongkannya jika cuti/sakit dianggap override kehadiran
					// Misalnya: CheckIn: "", CheckOut: "", // Jika cuti/sakit selalu mengoverride hadir
				}
				_, err := h.attendanceRepo.UpdateAttendance(c.Context(), existingAttendance.ID, &updatePayload) // Asumsi ada method UpdateAttendance yang menerima ID dan payload update
				if err != nil {
					log.Printf("ERROR: Gagal memperbarui absensi existing untuk tanggal %s: %v", currentDateStr, err)
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": fmt.Sprintf("Gagal memperbarui absensi tanggal %s: %v", currentDateStr, err),
					})
				}
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Status pengajuan berhasil diperbarui"})
}
