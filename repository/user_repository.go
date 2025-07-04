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
	"Sistem-Manajemen-Karyawan/models" // Pastikan ini 'models' jika folder Anda bernama 'models'
)

// UserRepository menyediakan metode untuk berinteraksi dengan koleksi users.
type UserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository membuat instance UserRepository baru.
func NewUserRepository() *UserRepository {
	return &UserRepository{
		collection: config.GetCollection(config.UserCollection), // Pastikan ini sesuai dengan definisi di config/database.go
	}
}

// CreateUser menambahkan user baru ke database.
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

// FindUserByEmail mencari user berdasarkan alamat email.
func (r *UserRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	filter := bson.M{"email": email}

	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // User not found, return nil user, nil error
		}
		return nil, fmt.Errorf("gagal menemukan user berdasarkan email: %w", err)
	}
	return &user, nil
}

// FindUserByID mencari user berdasarkan ObjectID.
func (r *UserRepository) FindUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	filter := bson.M{"_id": id}

	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // User not found, return nil user, nil error
		}
		return nil, fmt.Errorf("gagal menemukan user berdasarkan ID: %w", err)
	}
	return &user, nil
}

// UpdateUser memperbarui user berdasarkan ObjectID.
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

// DeleteUser menghapus user berdasarkan ObjectID.
func (r *UserRepository) DeleteUser(ctx context.Context, id primitive.ObjectID) (*mongo.DeleteResult, error) {
	filter := bson.M{"_id": id}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("gagal menghapus user: %w", err)
	}
	return result, nil
}

// GetAllUsers mengambil daftar semua user dengan opsi pagination dan filter.
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

// UpdateUserPassword memperbarui password user berdasarkan ObjectID.
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

// UpdateUserFirstLoginStatus memperbarui status is_first_login user.
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

// GetDashboardStats menghitung dan mengembalikan statistik dashboard
func (r *UserRepository) GetDashboardStats(ctx context.Context) (*models.DashboardStats, error) {
	// Total Karyawan
	totalUsers, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung total karyawan: %w", err)
	}

	// Karyawan Aktif (asumsi: role bukan 'admin' dan tidak dalam status 'cuti')
	activeUsers, err := r.collection.CountDocuments(ctx, bson.M{"role": "karyawan"})
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung karyawan aktif: %w", err)
	}

	// Karyawan Cuti (placeholder)
	leaveUsers := int64(0) 

	// Posisi Baru (asumsi: user yang dibuat dalam 30 hari terakhir)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	newPositions, err := r.collection.CountDocuments(ctx, bson.M{"created_at": bson.M{"$gte": thirtyDaysAgo}})
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung posisi baru: %w", err)
	}

	// Distribusi Departemen
	pipeline := []bson.M{
		{"$match": bson.M{"department": bson.M{"$ne": ""}}}, // Hanya departemen yang tidak kosong
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

	// Aktivitas Terbaru (placeholder)
	latestActivities := []string{
		"Sistem HR-System dimulai.",
		"Admin login ke dashboard.",
	}

	stats := &models.DashboardStats{
		TotalKaryawan:       totalUsers,
		KaryawanAktif:       activeUsers,
		KaryawanCuti:        leaveUsers,
		PosisiBaru:          newPositions,
		DistribusiDepartemen: departmentCounts,
		AktivitasTerbaru:    latestActivities,
	}

	return stats, nil
}
