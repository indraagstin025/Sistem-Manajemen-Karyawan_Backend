package handlers

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo" // Pastikan ini diimpor untuk mongo.Pipeline dan mongo.Cursor

	"Sistem-Manajemen-Karyawan/models"
	util "Sistem-Manajemen-Karyawan/pkg/utils"
	"Sistem-Manajemen-Karyawan/repository"
)

type UserHandler struct {
	userRepo   *repository.UserRepository
	deptRepo   repository.DepartmentRepository    // BARU: Tambahkan DepartmentRepository
	leaveRepo  repository.LeaveRequestRepository  // BARU: Tambahkan LeaveRequestRepository
	// Anda mungkin juga perlu AttendanceRepository jika ingin menghitung 'KaryawanCuti' dari absensi
	// attendanceRepo *repository.AttendanceRepository
}

// Perbarui konstruktor untuk menginisialisasi semua repository yang dibutuhkan.
// Pastikan parameter NewUserHandler ini sesuai dengan bagaimana Anda menginisialisasi
// repository di file main.go Anda.
func NewUserHandler(
	userRepo *repository.UserRepository,
	deptRepo repository.DepartmentRepository,     // Tambahkan parameter ini
	leaveRepo repository.LeaveRequestRepository,  // Tambahkan parameter ini
	// attendanceRepo *repository.AttendanceRepository, // Tambahkan ini jika dibutuhkan
) *UserHandler {
	return &UserHandler{
		userRepo:   userRepo,
		deptRepo:   deptRepo,   // Inisialisasi
		leaveRepo:  leaveRepo,  // Inisialisasi
		// attendanceRepo: attendanceRepo, // Inisialisasi
	}
}

// GetUserByID godoc
// @Summary Get User by ID
// @Description Mendapatkan detail user berdasarkan ID (user hanya bisa melihat data diri sendiri, admin bisa melihat semua)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} models.User "User berhasil ditemukan"
// @Failure 400 {object} object{error=string} "Invalid user ID format"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak - hanya bisa melihat data sendiri"
// @Failure 404 {object} object{error=string} "User tidak ditemukan"
// @Failure 500 {object} object{error=string} "Internal server error"
// @Router /users/{id} [get]
func (h *UserHandler) GetUserByID(c *fiber.Ctx) error {
    idParam := c.Params("id")
    objID, err := primitive.ObjectIDFromHex(idParam)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "format ID user tidak valid"})
    }

    claims, ok := c.Locals("user").(*models.Claims)
    if !ok {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "tidak terautentikasi atau klaim token tidak valid"})
    }

    if claims.Role != "admin" && claims.UserID.Hex() != idParam {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "akses ditolak. anda hanya dapat melihat profile anda sendiri."})
    }

    ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
    defer cancel()

    user, err := h.userRepo.FindUserByID(ctx, objID) // Baris ini akan mengembalikan (nil, nil) jika tidak ditemukan

    // >>>>> INI KODE YANG WAJIB ANDA TAMBAHKAN/PASTIKAN ADA <<<<<
    // Periksa apakah user ditemukan (tidak nil) DAN tidak ada error yang menunjukkan 'tidak ditemukan'
    if user == nil {
        if err == nil || err == mongo.ErrNoDocuments { // Jika user nil dan error juga nil, atau errornya ErrNoDocuments
            return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user tidak ditemukan"})
        }
        // Jika user nil tapi ada error lain (selain ErrNoDocuments), ini masalah di repo
        log.Printf("ERROR: FindUserByID mengembalikan user nil dengan error: %v", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal mendapatkan user (data kosong atau error repo)."})
    }
    // >>>>> AKHIR KODE YANG WAJIB ADA <<<<<


    if err != nil { // Blok ini akan menangani error lain dari repo (selain ErrNoDocuments)
        log.Printf("Error getting user by ID: %v", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mendapatkan user: %v", err)})
    }

    user.Password = "" // Baris ini sekarang aman karena 'user' dipastikan tidak nil
    return c.Status(fiber.StatusOK).JSON(user)
}

// GetAllUsers godoc
// @Summary Get All Users
// @Description Mendapatkan semua data users dengan pagination dan filter (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Param search query string false "Search by name or email"
// @Param role query string false "Filter by role"
// @Success 200 {object} object{data=array,total=int,page=int,limit=int} "Data users berhasil diambil"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak - hanya admin"
// @Failure 500 {object} object{error=string} "Gagal mengambil data users"
// @Router /admin/users [get]
func (h *UserHandler) GetAllUsers(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	search := c.Query("search", "")
	role := c.Query("role", "")

	if page < 1 {
		page = 1
	}

	if limit < 1 || limit > 100 {
		limit = 10
	}

	filter := bson.M{}
	if search != "" {
		filter["$or"] = []bson.M{
			{"name": primitive.Regex{Pattern: search, Options: "i"}},
			{"email": primitive.Regex{Pattern: search, Options: "i"}},
		}
	}
	if role != "" {
		filter["role"] = role
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	users, total, err := h.userRepo.GetAllUsers(ctx, filter, int64(page), int64(limit))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mendapatkan semua user: %v", err)})
	}
	for i := range users {
		users[i].Password = ""
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":  users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// UpdateUser godoc
// @Summary Update User
// @Description Update data user (user hanya bisa update data diri sendiri, admin bisa update semua)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param user body models.UserUpdatePayload true "Data update user"
// @Success 200 {object} object{message=string} "User berhasil diupdate"
// @Failure 400 {object} object{error=string,errors=array} "Invalid request body, user ID, atau validation error"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak - hanya bisa update data sendiri"
// @Failure 404 {object} object{message=string} "User tidak ditemukan"
// @Failure 500 {object} object{error=string} "Internal server error"
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "format ID user tidak valid"})
	}

	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "tidak terautentikasi atau klaim token tidak valid"})
	}

	if claims.Role != "admin" && claims.UserID.Hex() != idParam {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "akses ditolak. anda hanya dapat mengupdate profil anda sendiri."})
	}

	var payload models.UserUpdatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body", "details": err.Error()})
	}

	if errors := util.ValidateStruct(payload); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": errors})
	}

	updateData := bson.M{}

	if claims.Role != "admin" {
		if payload.Photo != "" {
			updateData["photo"] = payload.Photo
		}
		if payload.Address != "" {
			updateData["address"] = payload.Address
		}

		if payload.Name != "" || payload.Email != "" ||
			payload.Position != "" || payload.Department != "" || payload.BaseSalary != 0 {

			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "akses ditolak. anda tidak diizinkan mengubah nama, email, posisi, departemen, atau gaji dasar.",
			})
		}
	} else {
		if payload.Name != "" {
			updateData["name"] = payload.Name
		}
		if payload.Email != "" {
			updateData["email"] = payload.Email
		}
		if payload.Position != "" {
			updateData["position"] = payload.Position
		}
		if payload.Department != "" {
			updateData["department"] = payload.Department
		}
		if payload.BaseSalary != 0 {
			updateData["base_salary"] = payload.BaseSalary
		}
		if payload.Address != "" {
			updateData["address"] = payload.Address
		}
		if payload.Photo != "" {
			updateData["photo"] = payload.Photo
		}
	}

	if len(updateData) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tidak ada field yang akan diupdate"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	result, err := h.userRepo.UpdateUser(ctx, objID, updateData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mengupdate user: %v", err)})
	}
	if result.ModifiedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "user tidak ditemukan atau tidak ada perubahan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "user berhasil diupdate"})
}

// DeleteUser godoc
// @Summary Delete User
// @Description Menghapus user berdasarkan ID (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} object{message=string} "User berhasil dihapus"
// @Failure 400 {object} object{error=string} "Invalid ID format"
// @Failure 404 {object} object{message=string} "User tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal menghapus user"
// @Router /admin/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "format ID user tidak valid"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	result, err := h.userRepo.DeleteUser(ctx, objID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal menghapus user: %v", err)})
	}
	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "user tidak ditemukan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "user berhasil dihapus"})
}

// GetDashboardStats godoc
// @Summary Get Dashboard Statistics
// @Description Mendapatkan berbagai statistik untuk dashboard admin (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.DashboardStats "Statistik dashboard berhasil diambil"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak - hanya admin"
// @Failure 500 {object} object{error=string} "Gagal mengambil statistik dashboard"
// @Router /admin/dashboard-stats [get]
func (h *UserHandler) GetDashboardStats(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second) // Tambahkan timeout yang cukup
    defer cancel()

    // 1. Total Karyawan (total user di sistem)
    totalUsers, err := h.userRepo.CountDocuments(ctx, bson.M{})
    if err != nil {
        log.Printf("Error menghitung total user: %v", err) // Log error
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghitung total user."})
    }

    // 2. Karyawan Aktif (user dengan role "karyawan")
    activeUsers, err := h.userRepo.CountDocuments(ctx, bson.M{"role": "karyawan"})
    if err != nil {
        log.Printf("Error menghitung karyawan aktif: %v", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghitung karyawan aktif."})
    }

    // 3. Jumlah Karyawan Sedang Cuti (Placeholder, sesuaikan dengan logika riil Anda)
    // Jika Anda ingin menghitung ini dari Attendance, pastikan Anda punya attendanceRepo di handler ini
    karyawanCuti := int64(0) 

    // 4. Pengajuan Cuti/Izin Tertunda
    // Memastikan h.leaveRepo ada dan method CountPendingRequests tersedia
    pendingLeavesCount, err := h.leaveRepo.CountPendingRequests(ctx)
    if err != nil {
        log.Printf("Error menghitung pengajuan tertunda: %v", err)
        pendingLeavesCount = 0 // Tetapkan 0 jika terjadi error
    }

    // 5. Posisi Baru (karyawan yang dibuat dalam 30 hari terakhir)
    thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
    newPositions, err := h.userRepo.CountDocuments(ctx, bson.M{"created_at": bson.M{"$gte": thirtyDaysAgo}})
    if err != nil {
        log.Printf("Error menghitung posisi baru: %v", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghitung posisi baru."})
    }

    // 6. Distribusi Departemen (Total Departemen dan hitungan per departemen)
    // Memastikan h.deptRepo ada dan method CountDocuments tersedia
    totalDepartemen, err := h.deptRepo.CountDocuments(ctx, bson.M{}) // Asumsi ada CountDocuments di deptRepo
    if err != nil {
        log.Printf("Error menghitung total departemen: %v", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghitung total departemen."})
    }

    pipeline := mongo.Pipeline{ 
        bson.D{{Key: "$match", Value: bson.D{{Key: "department", Value: bson.D{{Key: "$ne", Value: ""}}}}}},
        bson.D{{Key: "$group", Value: bson.D{
            {Key: "_id",    Value: "$department"},
            {Key: "count",  Value: bson.D{{Key: "$sum", Value: 1}}},
        }}},
        bson.D{{Key: "$project", Value: bson.D{
            {Key: "department", Value: "$_id"},
            {Key: "count",      Value: 1},
            {Key: "_id",        Value: 0},
        }}},
    }

    cursor, err := h.userRepo.Aggregate(ctx, pipeline)
    if err != nil {
        log.Printf("Error melakukan agregasi distribusi departemen: %v", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal melakukan agregasi distribusi departemen."})
    }
    defer cursor.Close(ctx)

    var departmentDistribution []models.DepartmentCount
    if err = cursor.All(ctx, &departmentDistribution); err != nil {
        log.Printf("Error mendecode distribusi departemen: %v", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mendecode distribusi departemen."})
    }

    // 7. Aktivitas Terbaru (contoh statis)
    latestActivities := []string{
        "Sistem HR-System dimulai.",
        "Admin login ke dashboard.",
    }

    stats := &models.DashboardStats{
        TotalKaryawan:             totalUsers,
        KaryawanAktif:             activeUsers,
        KaryawanCuti:              karyawanCuti,
        PendingLeaveRequestsCount: pendingLeavesCount,
        PosisiBaru:                newPositions,
        TotalDepartemen:           totalDepartemen,
        DistribusiDepartemen:      departmentDistribution,
        AktivitasTerbaru:          latestActivities,
    }

    return c.Status(fiber.StatusOK).JSON(stats)
}

// UploadProfilePhoto godoc
// @Summary Upload User Profile Photo
// @Description Mengunggah foto profil untuk user tertentu. Hanya admin atau user itu sendiri yang bisa mengunggah.
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param photo formData file true "File foto profil (JPG, PNG, GIF, WEBP, maks 5MB)"
// @Success 200 {object} object{message=string,photo_url=string} "Foto profil berhasil diunggah"
// @Failure 400 {object} object{error=string} "Invalid file format, file size, atau no file uploaded"
// @Failure 401 {object} object{error=string} "Tidak terautentikasi"
// @Failure 403 {object} object{error=string} "Akses ditolak"
// @Failure 404 {object} object{message=string} "User tidak ditemukan"
// @Failure 500 {object} object{error=string} "Internal server error"
// @Router /users/{id}/upload-photo [post]
func (h *UserHandler) UploadProfilePhoto(c *fiber.Ctx) error {
	userID := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format ID user tidak valid"})
	}

	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tidak terautentikasi atau klaim token tidak valid"})
	}

	if claims.Role != "admin" && claims.UserID.Hex() != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak. Anda hanya dapat mengunggah foto profil Anda sendiri."})
	}

	file, err := c.FormFile("photo")
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tidak ada file foto yang diunggah."})
		}

		log.Printf("Error mengambil file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil file."})
	}

	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !allowedTypes[file.Header.Get("Content-Type")] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format file tidak didukung. Hanya JPG, PNG, GIF, WEBP yang diizinkan."})
	}

	const maxFileSize = 5 * 1024 * 1024
	if file.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Ukuran file terlalu besar. Maksimal %d MB.", maxFileSize/1024/1024)})
	}

	uploadDir := "./uploads"
	fileName := fmt.Sprintf("%s_%d%s", userID, time.Now().Unix(), filepath.Ext(file.Filename))
	filePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveFile(file, filePath); err != nil {
		log.Printf("Error menyimpan file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file foto."})
	}

	photoURL := fmt.Sprintf("http://localhost:3000/uploads/%s", fileName)

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	updateData := bson.M{"photo": photoURL}
	result, err := h.userRepo.UpdateUser(ctx, objID, updateData)
	if err != nil {
		log.Printf("Error mengupdate URL foto di database: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui URL foto di database."})
	}
	if result.ModifiedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User tidak ditemukan atau foto tidak berubah."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Foto profil berhasil diunggah.",
		"photo_url": photoURL,
	})
}