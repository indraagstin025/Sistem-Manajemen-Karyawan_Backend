package repository

import (
	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type WorkScheduleRepository struct {
	Collection *mongo.Collection
}

func NewWorkScheduleRepository() *WorkScheduleRepository {
	coll := config.GetCollection(config.WorkScheduleCollection)
	return &WorkScheduleRepository{
		Collection: coll,
	}
}

func (r *WorkScheduleRepository) Create(schedule *models.WorkSchedule) (*models.WorkSchedule, error) {
	schedule.ID = primitive.NewObjectID()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	_, err := r.Collection.InsertOne(context.TODO(), schedule)
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
	err := r.Collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *WorkScheduleRepository) FindByDate(date string) ([]*models.WorkSchedule, error) {
	filter := bson.M{"date": date}

	cursor, err := r.Collection.Find(context.TODO(), filter)
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
	cursor, err := r.Collection.Find(context.TODO(), filter)
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

func (r *WorkScheduleRepository) FindAllWithFilter(filter bson.M) ([]models.WorkSchedule, error) {
	cursor, err := r.Collection.Find(context.TODO(), filter)
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

	result, err := r.Collection.UpdateByID(context.TODO(), id, update)
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
	cursor, err := r.Collection.Find(context.TODO(), filter)
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
	res, err := r.Collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("jadwal tidak ditemukan")
	}
	return nil
}
