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
}

type attendanceRepository struct {
    qrCollection         *mongo.Collection
    attendanceCollection *mongo.Collection
    userCollection       *mongo.Collection
}

// NewAttendanceRepository menginisialisasi repository kehadiran.
func NewAttendanceRepository() AttendanceRepository {
    return &attendanceRepository{
        qrCollection:         config.GetCollection(config.QRCodeCollection),
        attendanceCollection: config.GetCollection(config.AttendanceCollection),
        userCollection:       config.GetCollection(config.UserCollection),
    }
}

// --- Implementasi untuk model.QRCode ---

func (r *attendanceRepository) CreateQRCode(ctx context.Context, qrCode *models.QRCode) (*mongo.InsertOneResult, error) {
    res, err := r.qrCollection.InsertOne(ctx, qrCode)
    if err != nil {
        return nil, fmt.Errorf("gagal membuat QR Code: %w", err)
    }
    return res, nil
}

func (r *attendanceRepository) FindQRCodeByValue(ctx context.Context, code string) (*models.QRCode, error) {
    var qrCode models.QRCode
    filter := bson.M{"code": code} // <-- Variabel filter DEKLARASI dan DIGUNAKAN
    err := r.qrCollection.FindOne(ctx, filter).Decode(&qrCode)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return nil, errors.New("QR Code tidak ditemukan.")
        }
        return nil, fmt.Errorf("gagal mencari QR Code berdasarkan nilai: %w", err)
    }
    return &qrCode, nil
}

func (r *attendanceRepository) FindActiveQRCodeByDate(ctx context.Context, date string) (*models.QRCode, error) {
    var qrCode models.QRCode

    filter := bson.M{
        "date": date,
        "expires_at": bson.M{"$gt": time.Now()}, // <<< Ini yang penting
    }

    opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}) 

    err := r.qrCollection.FindOne(ctx, filter, opts).Decode(&qrCode)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return nil, nil // Return nil, nil jika tidak ditemukan
        }
        return nil, fmt.Errorf("gagal mencari QR Code aktif: %w", err)
    }
    return &qrCode, nil
}

func (r *attendanceRepository) MarkQRCodeAsUsed(ctx context.Context, qrCodeID primitive.ObjectID, userID primitive.ObjectID) (*mongo.UpdateResult, error) {
    filter := bson.M{"_id": qrCodeID}
    update := bson.M{
        "$addToSet": bson.M{"used_by": userID},
        "$set":      bson.M{"updated_at": time.Now()},
    }

    res, err := r.qrCollection.UpdateOne(ctx, filter, update)
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
    filter := bson.M{"user_id": userID, "date": date} // <-- Variabel filter DEKLARASI dan DIGUNAKAN
    err := r.attendanceCollection.FindOne(ctx, filter).Decode(&attendance)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return nil, nil // Mengembalikan nil, nil jika tidak ditemukan
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
            {Key: "user_photo", Value: "$userDetails.photo"}, // Asumsi field photo di user model
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
    filter := bson.M{"user_id": userID} // <-- Variabel filter DEKLARASI dan DIGUNAKAN

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