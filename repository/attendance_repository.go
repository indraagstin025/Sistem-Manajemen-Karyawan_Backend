package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"


	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)


type AttendanceRepository interface {
	// --- Methods for QRCode ---
	CreateQRCode(ctx context.Context, qrCode *models.QRCode) (*mongo.InsertOneResult, error)
	FindQRCodeByValue(ctx context.Context, code string) (*models.QRCode, error)
	FindActiveQRCodeByDate(ctx context.Context, date string) (*models.QRCode, error)
	MarkQRCodeAsUsed(ctx context.Context, qrCodeID primitive.ObjectID, userID primitive.ObjectID) (*mongo.UpdateResult, error)

	// --- Methods for Attendance ---
	CreateAttendance(ctx context.Context, attendance *models.Attendance) (*mongo.InsertOneResult, error)
	FindAttendanceByUserAndDate(ctx context.Context, userID primitive.ObjectID, date string) (*models.Attendance, error)
	UpdateAttendanceCheckout(ctx context.Context, attendanceID primitive.ObjectID, checkOutTime string) (*mongo.UpdateResult, error)
	GetTodayAttendanceWithUserDetails(ctx context.Context) ([]models.AttendanceWithUser, error)
	FindAttendanceByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Attendance, error)
    UpdateAttendance(ctx context.Context, id primitive.ObjectID, payload *models.AttendanceUpdatePayload) (*mongo.UpdateResult, error)
	GetAllAttendancesWithUserDetails(ctx context.Context, filter bson.M, page, limit int64) ([]models.AttendanceWithUser, int64, error)
MarkAbsentEmployeesAsAlpha(
        ctx context.Context,
        userRepo *UserRepository, // <--- TAMBAHKAN *
        workScheduleRepo *WorkScheduleRepository, // <--- TAMBAHKAN *
        leaveRequestRepo LeaveRequestRepository, // Tipe ini sudah benar karena menggunakan interface
    ) error
}



type attendanceRepository struct {
	qrCodeCollection     *mongo.Collection
	attendanceCollection *mongo.Collection
	userCollection       *mongo.Collection
}

func NewAttendanceRepository() AttendanceRepository {
	return &attendanceRepository{
		qrCodeCollection:     config.GetCollection(config.QRCodeCollection),
		attendanceCollection: config.GetCollection(config.AttendanceCollection),
		userCollection:       config.GetCollection(config.UserCollection),
	}
}


func (r *attendanceRepository) CreateQRCode(ctx context.Context, qrCode *models.QRCode) (*mongo.InsertOneResult, error) {
	res, err := r.qrCodeCollection.InsertOne(ctx, qrCode)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat QR Code: %w", err)
	}
	return res, nil
}

func (r *attendanceRepository) FindQRCodeByValue(ctx context.Context, value string) (*models.QRCode, error) {
	var qrCode models.QRCode
	err := r.qrCodeCollection.FindOne(ctx, bson.M{"code": value}).Decode(&qrCode)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &qrCode, nil
}

func (r *attendanceRepository) FindActiveQRCodeByDate(ctx context.Context, date string) (*models.QRCode, error) {
	var qrCode models.QRCode

	filter := bson.M{
		"date":       date,
		"expires_at": bson.M{"$gt": time.Now()},
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})

	err := r.qrCodeCollection.FindOne(ctx, filter, opts).Decode(&qrCode)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil 
		}
		return nil, fmt.Errorf("gagal mencari QR Code aktif: %w", err)
	}
	return &qrCode, nil
}




func (r *attendanceRepository) GetAllAttendancesWithUserDetails(ctx context.Context, filter bson.M, page, limit int64) ([]models.AttendanceWithUser, int64, error) {
	
	total, err := r.attendanceCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung total dokumen absensi: %w", err)
	}


	findOptions := options.Find()
	findOptions.SetSkip((page - 1) * limit)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: "date", Value: -1}, {Key: "check_in", Value: -1}}) // Urutkan berdasarkan tanggal terbaru, lalu check-in

	
	pipeline := mongo.Pipeline{
		
		{{Key: "$match", Value: filter}},
		{{Key: "$sort", Value: bson.D{{Key: "date", Value: -1}, {Key: "check_in", Value: -1}}}},
		{{Key: "$skip", Value: (page - 1) * limit}},
		{{Key: "$limit", Value: limit}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: config.UserCollection},
			{Key: "localField", Value: "user_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "userDetails"},
		}}},
		
		{{Key: "$unwind", Value: "$userDetails"}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "id", Value: "$_id"}, 
			{Key: "user_id", Value: 1},
			{Key: "date", Value: 1},
			{Key: "check_in", Value: 1},
			{Key: "check_out", Value: 1},
			{Key: "status", Value: 1},
			{Key: "note", Value: 1},
			{Key: "user_name", Value: "$userDetails.name"},
			{Key: "user_email", Value: "$userDetails.email"},
			{Key: "user_photo", Value: "$userDetails.photo"},        
			{Key: "user_position", Value: "$userDetails.position"},
			{Key: "user_department", Value: "$userDetails.department"},
		}}},
	}

	cursor, err := r.attendanceCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal aggregation untuk riwayat kehadiran admin: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.AttendanceWithUser
	if err = cursor.All(ctx, &results); err != nil {
		return nil, 0, fmt.Errorf("gagal decode hasil aggregation riwayat kehadiran: %w", err)
	}

	if len(results) == 0 {
		return []models.AttendanceWithUser{}, total, nil // Pastikan mengembalikan slice kosong dan total
	}
	return results, total, nil
}


func (r *attendanceRepository) MarkQRCodeAsUsed(ctx context.Context, qrCodeID primitive.ObjectID, userID primitive.ObjectID) (*mongo.UpdateResult, error) {
	filter := bson.M{"_id": qrCodeID}
	update := bson.M{
		"$addToSet": bson.M{"used_by": userID},
		"$set":      bson.M{"updated_at": time.Now()},
	}

	res, err := r.qrCodeCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("gagal menandai QR Code sebagai sudah digunakan: %w", err)
	}
	return res, nil
}

func (r *attendanceRepository) CreateAttendance(ctx context.Context, attendance *models.Attendance) (*mongo.InsertOneResult, error) {
	res, err := r.attendanceCollection.InsertOne(ctx, attendance)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat absensi: %w", err)
	}
	return res, nil
}

func (r *attendanceRepository) FindAttendanceByUserAndDate(ctx context.Context, userID primitive.ObjectID, date string) (*models.Attendance, error) {
	var attendance models.Attendance
	filter := bson.M{"user_id": userID, "date": date}
	err := r.attendanceCollection.FindOne(ctx, filter).Decode(&attendance)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil 
		}
		return nil, fmt.Errorf("gagal mencari absensi berdasarkan user dan tanggal: %w", err)
	}
	return &attendance, nil
}

func (r *attendanceRepository) UpdateAttendanceCheckout(ctx context.Context, attendanceID primitive.ObjectID, checkOutTime string) (*mongo.UpdateResult, error) {
	update := bson.M{
		"$set": bson.M{
			"check_out":  checkOutTime,
			"updated_at": time.Now(),
		},
	}
	res, err := r.attendanceCollection.UpdateByID(ctx, attendanceID, update)
	if err != nil {
		return nil, fmt.Errorf("gagal update check-out absensi: %w", err)
	}
	return res, nil
}



func (r *attendanceRepository) GetTodayAttendanceWithUserDetails(ctx context.Context) ([]models.AttendanceWithUser, error) {

    wib, err := time.LoadLocation("Asia/Jakarta")
    if err != nil {

        return nil, fmt.Errorf("gagal memuat zona waktu Asia/Jakarta: %w", err)
    }
   
    today := time.Now().In(wib).Format("2006-01-02")

	fmt.Printf("✅ [DEBUG] Mengambil data absensi untuk tanggal: %s\n", today)

    pipeline := mongo.Pipeline{
        {{Key: "$match", Value: bson.D{{Key: "date", Value: today}}}},
        {{Key: "$lookup", Value: bson.D{
            {Key: "from", Value: config.UserCollection},
            {Key: "localField", Value: "user_id"},
            {Key: "foreignField", Value: "_id"},
            {Key: "as", Value: "userDetails"},
        }}},
        {{Key: "$unwind", Value: "$userDetails"}},
        {{Key: "$project", Value: bson.D{
            {Key: "_id", Value: "$_id"},
            {Key: "id", Value: "$_id"},
            {Key: "user_id", Value: 1},
            {Key: "date", Value: 1},
            {Key: "check_in", Value: 1},
            {Key: "check_out", Value: 1},
            {Key: "status", Value: 1},
            {Key: "note", Value: 1},
            {Key: "user_name", Value: "$userDetails.name"},
            {Key: "user_email", Value: "$userDetails.email"},
            {Key: "user_photo", Value: "$userDetails.photo"},
            {Key: "user_position", Value: "$userDetails.position"},
            {Key: "user_department", Value: "$userDetails.department"},
        }}},
    }

    cursor, err := r.attendanceCollection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, fmt.Errorf("gagal aggregation untuk daftar kehadiran hari ini: %w", err)
    }
    defer cursor.Close(ctx)

    var results []models.AttendanceWithUser
    if err = cursor.All(ctx, &results); err != nil {
        return nil, fmt.Errorf("gagal decode hasil aggregation kehadiran: %w", err)
    }

    if len(results) == 0 {
        return []models.AttendanceWithUser{}, nil
    }
    return results, nil
}

func (r *attendanceRepository) FindAttendanceByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Attendance, error) {
	filter := bson.M{"user_id": userID}

	cursor, err := r.attendanceCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("gagal mencari riwayat absensi user: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.Attendance
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("gagal decode riwayat absensi: %w", err)
	}

	if len(results) == 0 {
		return []models.Attendance{}, nil
	}
	return results, nil
}

// Dalam attendance_repository.go (atau AttendanceRepository interface)
func (r *attendanceRepository) UpdateAttendance(ctx context.Context, id primitive.ObjectID, payload *models.AttendanceUpdatePayload) (*mongo.UpdateResult, error) {
	update := bson.M{"$set": bson.M{}}
	if payload.CheckIn != "" {
		update["$set"].(bson.M)["check_in"] = payload.CheckIn
	}
	if payload.CheckOut != "" {
		update["$set"].(bson.M)["check_out"] = payload.CheckOut
	}
	if payload.Status != "" { // Pastikan ini juga diupdate
		update["$set"].(bson.M)["status"] = payload.Status
	}
	if payload.Note != "" { // Pastikan ini juga diupdate
		update["$set"].(bson.M)["note"] = payload.Note
	}
	update["$set"].(bson.M)["updated_at"] = time.Now()

	res, err := r.attendanceCollection.UpdateByID(ctx, id, update)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// file: repository/attendance_repository.go
// (Tambahkan di bagian bawah file)

// MarkAbsentEmployeesAsAlpha adalah fungsi yang akan dijalankan oleh cron job.
// Ia membutuhkan repository lain sebagai argumen untuk melakukan tugasnya.
// file: repository/attendance_repository.go

func (r *attendanceRepository) MarkAbsentEmployeesAsAlpha(
    ctx context.Context,
    userRepo *UserRepository,
    workScheduleRepo *WorkScheduleRepository,
    leaveRequestRepo LeaveRequestRepository,
) error {
    fmt.Println("🚀 [Cron Job] Memulai tugas: Menandai karyawan Alpha...")
    now := time.Now()
    today := now.Format("2006-01-02")

    activeUsers, err := userRepo.FindAllActiveUsers(ctx)
    if err != nil {
        fmt.Printf("❌ [Cron Job] Error mendapatkan user aktif: %v\n", err)
        return err
    }

    alphaCount := 0
    for _, user := range activeUsers {
        if user.Role == "admin" {
            continue
        }

        schedule, _ := workScheduleRepo.FindApplicableScheduleForUser(ctx, user.ID, today)
        if schedule == nil {
            continue 
        }

        // =======================================================
        // BARU: Cek apakah jam sekarang sudah melewati jam pulang
        // =======================================================
        wib, _ := time.LoadLocation("Asia/Jakarta")
        scheduleEndTime, err := time.ParseInLocation("15:04", schedule.EndTime, wib)
        if err != nil {
            continue // Lewati jika format EndTime salah
        }

        // Buat waktu EndTime hari ini secara lengkap
        fullScheduleEndTime := time.Date(now.Year(), now.Month(), now.Day(), scheduleEndTime.Hour(), scheduleEndTime.Minute(), 0, 0, wib)

        // Jika jam sekarang BELUM melewati jam pulang, jangan proses dulu
        if now.Before(fullScheduleEndTime) {
            continue
        }
        // =======================================================

        attendance, _ := r.FindAttendanceByUserAndDate(ctx, user.ID, today)
        if attendance != nil {
            continue
        }

        leave, _ := leaveRequestRepo.FindApprovedRequestByUserAndDate(ctx, user.ID, today)
        if leave != nil {
            continue
        }

        fmt.Printf("✔️ [Cron Job] Menandai user %s sebagai Alpha...\n", user.Name)
        alphaAttendance := &models.Attendance{
            ID:        primitive.NewObjectID(),
            UserID:    user.ID,
            Date:      today,
            Status:    "Alpha",
            Note:      "Dibuat otomatis oleh sistem (setelah jam kerja)",
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        }

        _, createErr := r.CreateAttendance(ctx, alphaAttendance)
        if createErr != nil {
            fmt.Printf("❌ [Cron Job] Gagal menyimpan data Alpha untuk user %s: %v\n", user.ID.Hex(), createErr)
        } else {
            alphaCount++
        }
    }

    fmt.Printf("✅ [Cron Job] Selesai. %d karyawan ditandai sebagai Alpha.\n", alphaCount)
    return nil
}