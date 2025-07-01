package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		collection: config.GetCollection(config.UserCollection),
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) (*mongo.InsertOneResult, error) {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsFirstLogin = true

	result, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, fmt.Errorf("email sudah ada")
		}
		return nil, fmt.Errorf("gagal membuat user: %v", err)
	}
	return result, nil
}

func (r *UserRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	filter := bson.M{"email": email}

	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user tidak ditemukan")
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	filter := bson.M{"_id": id}

	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {

		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user tidak ditemukan")
		}
		return nil, fmt.Errorf("gagal menemukan user berdasarkan ID: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, id primitive.ObjectID, updateData bson.M) (*mongo.UpdateResult, error) {
	updateData["updated_at"] = time.Now()
	filter := bson.M{"_id": id}
	update := bson.M{"$set": updateData}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("gagal mengupdate User: %w", err)
	}
	return result, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, id primitive.ObjectID) (*mongo.DeleteResult, error) {
	filter := bson.M{"_id": id}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}
	return result, nil
}

func (r *UserRepository) GetAllUsers(ctx context.Context, filter bson.M, page, limit int64) ([]models.User, int64, error) {
	findOptions := options.Find()
	findOptions.SetSkip((page - 1) * limit)
	findOptions.SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal menemukan users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User

	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, fmt.Errorf("gagal mendecode User: %w", err)
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal Menghitung user: %w", err)
	}

	return users, total, nil
}
