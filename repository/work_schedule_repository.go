package repository

import (
	"context"
	"errors"
	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var scheduleCollection = config.GetCollection(config.WorkScheduleCollection)

// WorkScheduleRepository struct kosong untuk mengimplementasikan method
type WorkScheduleRepository struct{}

// NewWorkScheduleRepository membuat instance baru dari WorkScheduleRepository
func NewWorkScheduleRepository() *WorkScheduleRepository {
	return &WorkScheduleRepository{}
}

func (r *WorkScheduleRepository) Create(schedule *models.WorkSchedule) (*models.WorkSchedule, error) {
	schedule.ID = primitive.NewObjectID()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	_, err := scheduleCollection.InsertOne(context.TODO(), schedule)
	if err != nil {
		return nil, err
	}
	return schedule, nil
}

func (r *WorkScheduleRepository) FindByUserAndDate(userID primitive.ObjectID, date string) (*models.WorkSchedule, error) {
	filter := bson.M{
		"user_id": userID,
		"date":    date,
	}

	var result models.WorkSchedule
	err := scheduleCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *WorkScheduleRepository) FindByDate(date string) ([]*models.WorkSchedule, error) {
	filter := bson.M{"date": date}

	cursor, err := scheduleCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var results []*models.WorkSchedule
	for cursor.Next(context.TODO()) {
		var s models.WorkSchedule
		if err := cursor.Decode(&s); err != nil {
			return nil, err
		}
		results = append(results, &s)
	}

	return results, nil
}

func (r *WorkScheduleRepository) FindByUser(userID primitive.ObjectID) ([]*models.WorkSchedule, error) {
	filter := bson.M{"user_id": userID}
	cursor, err := scheduleCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var results []*models.WorkSchedule
	for cursor.Next(context.TODO()) {
		var schedule models.WorkSchedule
		if err := cursor.Decode(&schedule); err != nil {
			return nil, err
		}
		results = append(results, &schedule)
	}
	return results, nil
}

// ✨ Tambahkan method ini untuk mendukung filtering dinamis di GetAllWorkSchedules handler
func (r *WorkScheduleRepository) FindAllWithFilter(filter bson.M) ([]models.WorkSchedule, error) {
	cursor, err := scheduleCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var schedules []models.WorkSchedule
	if err = cursor.All(context.TODO(), &schedules); err != nil {
		return nil, err
	}
	return schedules, nil
}


func (r *WorkScheduleRepository) UpdateByID(id primitive.ObjectID, payload *models.WorkScheduleUpdatePayload) error {
	update := bson.M{
		"$set": bson.M{
			"start_time": payload.StartTime,
			"end_time":   payload.EndTime,
			"note":       payload.Note,
			"updated_at": time.Now(),
		},
	}

	result, err := scheduleCollection.UpdateByID(context.TODO(), id, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("jadwal tidak ditemukan")
	}
	return nil
}

func (r *WorkScheduleRepository) FindByUserAndDateRange(userID primitive.ObjectID, startDate, endDate string) ([]*models.WorkSchedule, error) {
	filter := bson.M{
		"user_id": userID,
		"date": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	cursor, err := scheduleCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var results []*models.WorkSchedule
	for cursor.Next(context.TODO()) {
		var schedule models.WorkSchedule
		if err := cursor.Decode(&schedule); err != nil {
			return nil, err
		}
		results = append(results, &schedule)
	}
	return results, nil
}

func (r *WorkScheduleRepository) DeleteByID(id primitive.ObjectID) error {
	res, err := scheduleCollection.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("jadwal tidak ditemukan")
	}
	return nil
}