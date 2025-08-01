package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"	
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"
	"github.com/fogleman/gg"
	util "Sistem-Manajemen-Karyawan/pkg/utils"
	"Sistem-Manajemen-Karyawan/repository"
)

type UserHandler struct {
	userRepo  *repository.UserRepository
	deptRepo  repository.DepartmentRepository
	leaveRepo repository.LeaveRequestRepository
}

// Perbarui konstruktor untuk menginisialisasi semua repository yang dibutuhkan.
// Pastikan parameter NewUserHandler ini sesuai dengan bagaimana Anda menginisialisasi
// repository di file main.go Anda.
func NewUserHandler(
	userRepo *repository.UserRepository,
	deptRepo repository.DepartmentRepository,
	leaveRepo repository.LeaveRequestRepository,

) *UserHandler {
	return &UserHandler{
		userRepo:  userRepo,
		deptRepo:  deptRepo,
		leaveRepo: leaveRepo,
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

	user, err := h.userRepo.FindUserByID(ctx, objID)

	if user == nil {
		if err == nil || err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user tidak ditemukan"})
		}

		log.Printf("ERROR: FindUserByID mengembalikan user nil dengan error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal mendapatkan user (data kosong atau error repo)."})
	}

	if err != nil {
		log.Printf("Error getting user by ID: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("gagal mendapatkan user: %v", err)})
	}

	user.Password = ""
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
// @Success 200 {object} object{data=[]models.User,total=int,page=int,limit=int} "Data users berhasil diambil" // <-- Perbaikan di sini
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
// @Description Update data user (user hanya bisa update data diri sendiri, admin bisa update semua, karyawan sekarang bisa mengubah email sendiri)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param user body models.UserUpdatePayload true "Data update user"
// @Success 200 {object} object{message=string} "User berhasil diupdate"
// @Failure 400 {object} models.ValidationErrorResponse "Invalid request body, user ID, atau validation error" // <-- Perbaikan di sini
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

	// Memeriksa otorisasi: User yang login harus sama dengan user yang diupdate, ATAU user yang login harus admin.
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
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	// Logika update berdasarkan peran
	if claims.Role != "admin" {
		// Jika user bukan admin (karyawan), hanya izinkan update untuk Photo, Address, dan EMAIL
		if payload.Photo != "" {
			updateData["photo"] = payload.Photo
		}
		if payload.Address != "" {
			updateData["address"] = payload.Address
		}

		// Izinkan karyawan untuk mengubah email mereka sendiri, dengan validasi keunikan
		if payload.Email != "" {
			isEmailTaken, err := h.userRepo.IsEmailTaken(ctx, payload.Email, objID)
			if err != nil {
				log.Printf("Error checking email existence: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memeriksa ketersediaan email."})
			}
			if isEmailTaken {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email sudah digunakan oleh user lain."})
			}
			updateData["email"] = payload.Email
		}

		// Batasi perubahan lain untuk non-admin
		if payload.Name != "" ||
			payload.Position != "" || payload.Department != "" || payload.BaseSalary != 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "akses ditolak. anda tidak diizinkan mengubah nama, posisi, departemen, atau gaji dasar.",
			})
		}
	} else { // Jika user adalah admin, izinkan update semua bidang
		if payload.Name != "" {
			updateData["name"] = payload.Name
		}
		if payload.Email != "" {
			// Admin juga perlu validasi email unik, mengecualikan user yang sedang diupdate
			isEmailTaken, err := h.userRepo.IsEmailTaken(ctx, payload.Email, objID)
			if err != nil {
				log.Printf("Error checking email existence for admin update: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memeriksa ketersediaan email saat update admin."})
			}
			if isEmailTaken {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email sudah digunakan oleh user lain."})
			}
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

	totalUsers, err := h.userRepo.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("Error menghitung total user: %v", err) // Log error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghitung total user."})
	}

	activeUsers, err := h.userRepo.CountDocuments(ctx, bson.M{"role": "karyawan"})
	if err != nil {
		log.Printf("Error menghitung karyawan aktif: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghitung karyawan aktif."})
	}

	karyawanCuti := int64(0)

	pendingLeavesCount, err := h.leaveRepo.CountPendingRequests(ctx)
	if err != nil {
		log.Printf("Error menghitung pengajuan tertunda: %v", err)
		pendingLeavesCount = 0
	}

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	newPositions, err := h.userRepo.CountDocuments(ctx, bson.M{"created_at": bson.M{"$gte": thirtyDaysAgo}})
	if err != nil {
		log.Printf("Error menghitung posisi baru: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghitung posisi baru."})
	}

	totalDepartemen, err := h.deptRepo.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("Error menghitung total departemen: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghitung total departemen."})
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{{Key: "department", Value: bson.D{{Key: "$ne", Value: ""}}}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$department"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "department", Value: "$_id"},
			{Key: "count", Value: 1},
			{Key: "_id", Value: 0},
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tidak ada file foto yang diunggah."})
	}

	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	contentType := file.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format file tidak didukung. Hanya JPG, PNG, GIF, WEBP yang diizinkan."})
	}

	const maxFileSize = 5 * 1024 * 1024
	if file.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Ukuran file terlalu besar. Maksimal %d MB.", maxFileSize/1024/1024)})
	}

	// Buka file
	fileReader, err := file.Open()
	if err != nil {
		log.Printf("Gagal membuka file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuka file."})
	}
	defer fileReader.Close()

	// Ambil bucket GridFS dari config
	bucket, err := config.GetGridFSBucket()
	if err != nil {
		log.Printf("Gagal membuat GridFS bucket: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengakses GridFS bucket."})
	}

	// Upload file ke GridFS
	uploadStream, err := bucket.OpenUploadStream(file.Filename)
	if err != nil {
		log.Printf("Gagal membuka stream GridFS: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file ke GridFS."})
	}
	defer uploadStream.Close()

	_, err = io.Copy(uploadStream, fileReader)
	if err != nil {
		log.Printf("Gagal menulis ke stream GridFS: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file ke GridFS."})
	}

	photoID := uploadStream.FileID.(primitive.ObjectID)

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	updateData := bson.M{
		"photo_id":   photoID,
		"photo_mime": contentType,
	}

	result, err := h.userRepo.UpdateUser(ctx, objID, updateData)
	if err != nil {
		log.Printf("Error mengupdate data user: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui user."})
	}
	if result.ModifiedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User tidak ditemukan atau data tidak berubah."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Foto profil berhasil diunggah.",
		"photo_id":  photoID.Hex(),
		"mime_type": contentType,
	})
}

// GetProfilePhoto godoc
// @Summary Get User Profile Photo
// @Description Mengambil foto profil user berdasarkan ID. Jika tidak ada foto, akan mengembalikan placeholder default.
// @Tags Users
// @Accept json
// @Produce image/jpeg,image/png,image/gif,image/webp
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {file} file "Foto profil berhasil diambil atau placeholder default dikembalikan"
// @Failure 400 {object} object{error=string} "ID user tidak valid"
// @Failure 404 {object} object{error=string} "User tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal mengambil foto profil atau placeholder"
// @Router /users/{id}/photo [get]
func (h *UserHandler) GetProfilePhoto(c *fiber.Ctx) error {
	// Bagian validasi ID dan pencarian user tetap sama
	userID := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID user tidak valid"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	user, err := h.userRepo.FindUserByID(ctx, objID)
	if err != nil {
		log.Printf("Error finding user by ID %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data user"})
	}
	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User tidak ditemukan"})
	}

	// Jika PhotoID kosong (user belum upload foto)
	if user.PhotoID.IsZero() {
		// --- PERBAIKAN DIMULAI DI SINI ---

		// io.Pipe membuat pasangan reader (pr) dan writer (pw) yang terhubung.
		// Data yang ditulis ke 'pw' akan bisa dibaca dari 'pr'.
		pr, pw := io.Pipe()

		// Proses pembuatan gambar harus dijalankan di goroutine terpisah
		// agar tidak memblokir proses pengiriman data ke klien.
		go func() {
			// Penting: Pastikan untuk menutup writer setelah selesai,
			// ini akan memberi sinyal kepada reader bahwa data sudah berakhir.
			defer pw.Close()

			initial := "A"
			if user.Name != "" {
				initial = strings.ToUpper(string([]rune(user.Name)[0]))
			}

			dc := gg.NewContext(128, 128)
			dc.SetHexColor("#E2E8F0")
			dc.Clear()
			dc.SetHexColor("#4A5568")

			if err := dc.LoadFontFace("public/fonts/Inter-Bold.ttf", 64); err != nil {
				log.Printf("Could not load font file: %v", err)
			} else {
				dc.DrawStringAnchored(initial, 128/2, 128/2, 0.5, 0.5)
			}

			// Tulis gambar PNG yang sudah jadi ke dalam 'PipeWriter'.
			// Jika ada error saat menulis, tutup pipe dengan error tersebut.
			if err := dc.EncodePNG(pw); err != nil {
				pw.CloseWithError(err)
			}
		}()

		// Set header Content-Type
		c.Set("Content-Type", "image/png")

		// Kirim 'PipeReader' ke klien. SendStream akan membaca dari 'pr'
		// dan mengalirkannya ke browser hingga 'pr' ditutup.
		return c.SendStream(pr)
		// --- PERBAIKAN BERAKHIR DI SINI ---
	}

	// Bagian ini untuk mengambil foto dari GridFS, tidak ada perubahan
	bucket, err := config.GetGridFSBucket()
	if err != nil {
		log.Printf("Error accessing GridFS: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengakses penyimpanan file"})
	}

	var buf bytes.Buffer
	_, err = bucket.DownloadToStream(user.PhotoID, &buf)
	if err != nil {
		log.Printf("Error reading photo from GridFS for user %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membaca foto profil"})
	}

	contentType := "image/jpeg"
	if user.PhotoMime != "" {
		contentType = user.PhotoMime
	}
	c.Set("Content-Type", contentType)

	return c.Status(fiber.StatusOK).Send(buf.Bytes())
}
