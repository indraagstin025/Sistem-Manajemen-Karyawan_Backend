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

func (r *UserRepository) GetDashboardStats(ctx context.Context) (*models.DashboardStats, error) {

	totalUsers, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung total karyawan: %w", err)
	}

	activeUsers, err := r.collection.CountDocuments(ctx, bson.M{"role": "karyawan"})
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung karyawan aktif: %w", err)
	}

	leaveUsers := int64(0)

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	newPositions, err := r.collection.CountDocuments(ctx, bson.M{"created_at": bson.M{"$gte": thirtyDaysAgo}})
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung posisi baru: %w", err)
	}

	pipeline := []bson.M{
		{"$match": bson.M{"department": bson.M{"$ne": ""}}},
		{"$group": bson.M{
			"_id":   "$department",
			"count": bson.M{"$sum": 1},
		}},
		{"$project": bson.M{
			"department": "$_id",
			"count":      1,
			"_id":        0,
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("gagal melakukan agregasi distribusi departemen: %w", err)
	}
	defer cursor.Close(ctx)

	var departmentCounts []models.DepartmentCount
	if err = cursor.All(ctx, &departmentCounts); err != nil {
		return nil, fmt.Errorf("gagal mendecode distribusi departemen: %w", err)
	}

	latestActivities := []string{
		"Sistem HR-System dimulai.",
		"Admin login ke dashboard.",
	}

	stats := &models.DashboardStats{
		TotalKaryawan:        totalUsers,
		KaryawanAktif:        activeUsers,
		KaryawanCuti:         leaveUsers,
		PosisiBaru:           newPositions,
		DistribusiDepartemen: departmentCounts,
		AktivitasTerbaru:     latestActivities,
	}

	return stats, nil
}
