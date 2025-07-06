package repository

import (
	"context"
	"time"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AttendanceRepository interface {
	FindQRCodeByValue(code string) (*models.QRCode, error)
	FindAttendanceByUserAndDate(userID primitive.ObjectID, date string) (*models.Attendance, error)
	CreateAttendance(attendance *models.Attendance) (*mongo.InsertOneResult, error)
	UpdateAttendanceCheckout(attendanceID primitive.ObjectID, checkOutTime string) (*mongo.UpdateResult, error)
	MarkQRCodeAsUsed(qrCodeID primitive.ObjectID, userID primitive.ObjectID) (*mongo.UpdateResult, error)
	GetTodayAttendanceWithUserDetails() ([]models.AttendanceWithUser, error)
	CreateQRCode(qrCode *models.QRCode) (*mongo.InsertOneResult, error)
	FindAttendanceByUserID(userID primitive.ObjectID) ([]models.Attendance, error)
}

func (r *attendanceRepository) CreateQRCode(qrCode *models.QRCode) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.qrCollection.InsertOne(ctx, qrCode)
}

type attendanceRepository struct {
	qrCollection         *mongo.Collection
	attendanceCollection *mongo.Collection
}

func NewAttendanceRepository() AttendanceRepository {
	return &attendanceRepository{
		qrCollection:         config.GetCollection(config.QRCodeCollection),
		attendanceCollection: config.GetCollection(config.AttendanceCollection),
	}
}

func (r *attendanceRepository) FindQRCodeByValue(code string) (*models.QRCode, error) {
	var qrCode models.QRCode
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := r.qrCollection.FindOne(ctx, bson.M{"code": code}).Decode(&qrCode)
	if err != nil {
		return nil, err
	}
	return &qrCode, nil
}

func (r *attendanceRepository) FindAttendanceByUserAndDate(userID primitive.ObjectID, date string) (*models.Attendance, error) {
	var attendance models.Attendance
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := r.attendanceCollection.FindOne(ctx, bson.M{"user_id": userID, "date": date}).Decode(&attendance)
	if err != nil {
		return nil, err
	}
	return &attendance, nil
}

func (r *attendanceRepository) CreateAttendance(attendance *models.Attendance) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return r.attendanceCollection.InsertOne(ctx, attendance)
}

func (r *attendanceRepository) UpdateAttendanceCheckout(attendanceID primitive.ObjectID, checkOutTime string) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"check_out":  checkOutTime,
			"updated_at": time.Now(),
		},
	}
	return r.attendanceCollection.UpdateOne(ctx, bson.M{"_id": attendanceID}, update)
}

func (r *attendanceRepository) MarkQRCodeAsUsed(qrCodeID primitive.ObjectID, userID primitive.ObjectID) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{"$push": bson.M{"used_by": userID}}
	return r.qrCollection.UpdateOne(ctx, bson.M{"_id": qrCodeID}, update)
}

func (r *attendanceRepository) GetTodayAttendanceWithUserDetails() ([]models.AttendanceWithUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

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
			{Key: "_id", Value: 1},
			{Key: "user_id", Value: 1},
			{Key: "date", Value: 1},
			{Key: "check_in", Value: 1},
			{Key: "check_out", Value: 1},
			{Key: "status", Value: 1},
			{Key: "note", Value: 1},
			{Key: "user_name", Value: "$userDetails.name"},
			{Key: "user_email", Value: "$userDetails.email"},
			{Key: "user_photo", Value: "$userDetails.photo_url"},
		}}},
	}

	cursor, err := r.attendanceCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.AttendanceWithUser
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (r *attendanceRepository) FindAttendanceByUserID(userID primitive.ObjectID) ([]models.Attendance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}

	cursor, err := r.attendanceCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.Attendance
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}
