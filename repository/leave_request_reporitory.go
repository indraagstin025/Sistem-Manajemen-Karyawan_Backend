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

type LeaveRequestRepository interface {
	Create(req *models.LeaveRequest) (*mongo.InsertOneResult, error)
	FindAll() ([]models.LeaveRequest, error)
	FindByID(id primitive.ObjectID) (*models.LeaveRequest, error)
	UpdateStatus(id primitive.ObjectID, status string, note string) (*mongo.UpdateResult, error)
	UpdateAttachmentURL(id primitive.ObjectID, fileURL string) (*mongo.UpdateResult, error)
}

func (r *leaveRequestRepository) UpdateAttachmentURL(id primitive.ObjectID, fileURL string) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	update := bson.M{"$set": bson.M{"attachment_url": fileURL}}
	return r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
}

type leaveRequestRepository struct {
	collection *mongo.Collection
}

func NewLeaveRequestRepository() LeaveRequestRepository {
	return &leaveRequestRepository{
		collection: config.GetCollection(config.LeaveRequestCollection),
	}
}

func (r *leaveRequestRepository) Create(req *models.LeaveRequest) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.collection.InsertOne(ctx, req)
}

func (r *leaveRequestRepository) FindAll() ([]models.LeaveRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var requests []models.LeaveRequest
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &requests); err != nil {
		return nil, err
	}
	return requests, nil
}

func (r *leaveRequestRepository) FindByID(id primitive.ObjectID) (*models.LeaveRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request models.LeaveRequest
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&request)
	return &request, err
}

func (r *leaveRequestRepository) UpdateStatus(id primitive.ObjectID, status string, note string) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"note":       note,
			"updated_at": time.Now(),
		},
	}
	return r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
}