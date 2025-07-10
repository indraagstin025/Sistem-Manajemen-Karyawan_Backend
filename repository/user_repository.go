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
		return nil, fmt.Errorf("gagal membuat user: %w", err)
	}
	return result, nil
}

func (r *UserRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	filter := bson.M{"email": email}

	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal menemukan user berdasarkan email: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	filter := bson.M{"_id": id}

	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
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
		return nil, fmt.Errorf("gagal mengupdate user: %w", err)
	}
	return result, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, id primitive.ObjectID) (*mongo.DeleteResult, error) {
	filter := bson.M{"_id": id}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("gagal menghapus user: %w", err)
	}
	return result, nil
}

func (r *UserRepository) GetAllUsers(ctx context.Context, filter bson.M, page, limit int64) ([]models.User, int64, error) {
	findOptions := options.Find()
	findOptions.SetSkip((page - 1) * limit)
	findOptions.SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal menemukan user: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User

	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, fmt.Errorf("gagal mendecode user: %w", err)
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung user: %w", err)
	}

	return users, total, nil
}

func (r *UserRepository) UpdateUserPassword(ctx context.Context, id primitive.ObjectID, hashedPassword string) error {
	update := bson.M{
		"$set": bson.M{
			"password":   hashedPassword,
			"updated_at": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("gagal mengupdate password user: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateUserFirstLoginStatus(ctx context.Context, id primitive.ObjectID, status bool) error {
	update := bson.M{
		"$set": bson.M{
			"is_first_login": status,
			"updated_at":     time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("gagal mengupdate status first login: %w", err)
	}
	return nil
}



// CountDocuments adalah fungsi umum untuk menghitung dokumen berdasarkan filter
func (r *UserRepository) CountDocuments(ctx context.Context, filter bson.M) (int64, error) {
    count, err := r.collection.CountDocuments(ctx, filter)
    if err != nil {
        return 0, fmt.Errorf("gagal menghitung dokumen dari koleksi user: %w", err)
    }
    return count, nil
}

// Aggregate adalah fungsi umum untuk menjalankan pipeline agregasi pada koleksi user
func (r *UserRepository) Aggregate(ctx context.Context, pipeline mongo.Pipeline) (*mongo.Cursor, error) {
    cursor, err := r.collection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, fmt.Errorf("gagal menjalankan agregasi pada koleksi user: %w", err)
    }
    return cursor, nil
}