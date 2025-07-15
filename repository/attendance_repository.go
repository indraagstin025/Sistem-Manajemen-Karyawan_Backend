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

// AttendanceRepository mendefinisikan interface untuk operasi database terkait kehadiran dan QR Code.
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
}



type attendanceRepository struct {
	qrCodeCollection     *mongo.Collection
	attendanceCollection *mongo.Collection
	userCollection       *mongo.Collection
}

// NewAttendanceRepository menginisialisasi repository kehadiran.
func NewAttendanceRepository() AttendanceRepository {
	return &attendanceRepository{
		qrCodeCollection:     config.GetCollection(config.QRCodeCollection),
		attendanceCollection: config.GetCollection(config.AttendanceCollection),
		userCollection:       config.GetCollection(config.UserCollection),
	}
}

// --- Implementasi untuk model.QRCode ---

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
			return nil, nil // Return nil, nil jika tidak ditemukan
		}
		return nil, fmt.Errorf("gagal mencari QR Code aktif: %w", err)
	}
	return &qrCode, nil
}

// attendance/repository/attendance_repository.go

// ... (kode yang sudah ada sebelumnya) ...

// Implementasi metode baru
func (r *attendanceRepository) GetAllAttendancesWithUserDetails(ctx context.Context, filter bson.M, page, limit int64) ([]models.AttendanceWithUser, int64, error) {
	// Hitung total dokumen yang cocok dengan filter untuk pagination
	total, err := r.attendanceCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung total dokumen absensi: %w", err)
	}

	// Atur opsi untuk pagination
	findOptions := options.Find()
	findOptions.SetSkip((page - 1) * limit)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: "date", Value: -1}, {Key: "check_in", Value: -1}}) // Urutkan berdasarkan tanggal terbaru, lalu check-in

	// Pipeline agregasi untuk menggabungkan data absensi dengan detail user
	pipeline := mongo.Pipeline{
		// Tahap $match untuk filter yang diberikan
		{{Key: "$match", Value: filter}},
		// Tahap $sort untuk sorting (opsional, bisa juga di FindOptions)
		{{Key: "$sort", Value: bson.D{{Key: "date", Value: -1}, {Key: "check_in", Value: -1}}}},
		// Tahap $skip untuk pagination
		{{Key: "$skip", Value: (page - 1) * limit}},
		// Tahap $limit untuk pagination
		{{Key: "$limit", Value: limit}},
		// Tahap $lookup untuk menggabungkan dengan koleksi users
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: config.UserCollection},
			{Key: "localField", Value: "user_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "userDetails"},
		}}},
		// Tahap $unwind untuk mengeluarkan array userDetails menjadi objek tunggal
		{{Key: "$unwind", Value: "$userDetails"}},
		// Tahap $project untuk memilih dan membentuk ulang field yang ingin ditampilkan
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "id", Value: "$_id"}, // Memberikan ID sebagai string
			{Key: "user_id", Value: 1},
			{Key: "date", Value: 1},
			{Key: "check_in", Value: 1},
			{Key: "check_out", Value: 1},
			{Key: "status", Value: 1},
			{Key: "note", Value: 1},
			{Key: "user_name", Value: "$userDetails.name"},
			{Key: "user_email", Value: "$userDetails.email"},
			{Key: "user_photo", Value: "$userDetails.photo"},        // Asumsi 'photo' menyimpan URL atau ID foto
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

// ... (sisa kode attendanceRepository lainnya) ...

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

// --- Implementasi untuk model.Attendance ---

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
			return nil, nil // Mengembalikan nil, nil jika tidak ditemukan (sudah benar)
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
	today := time.Now().Format("2006-01-02")

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
