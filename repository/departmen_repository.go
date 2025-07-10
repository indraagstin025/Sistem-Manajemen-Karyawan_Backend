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

// Perbarui interface DepartmentRepository
type DepartmentRepository interface {
	CreateDepartment(ctx context.Context, department *models.Department) (*mongo.InsertOneResult, error)
	GetAllDepartments(ctx context.Context) ([]models.Department, error)
	GetDepartmentByID(ctx context.Context, id primitive.ObjectID) (*models.Department, error)
	UpdateDepartment(ctx context.Context, id primitive.ObjectID, updateData bson.M) (*mongo.UpdateResult, error)
	DeleteDepartment(ctx context.Context, id primitive.ObjectID) (*mongo.DeleteResult, error)
	FindDepartmentByName(ctx context.Context, name string) (*models.Department, error)
	CountDocuments(ctx context.Context, filter bson.M) (int64, error) // <--- BARU: Tambahkan method ini ke interface
}

type departmentRepository struct {
	collection *mongo.Collection
}

func NewDepartmentRepository() DepartmentRepository {
	return &departmentRepository{
		collection: config.GetCollection(config.DepartmentCollection),
	}
}

func (r *departmentRepository) CreateDepartment(ctx context.Context, department *models.Department) (*mongo.InsertOneResult, error) {
	department.ID = primitive.NewObjectID()
	department.CreatedAt = time.Now()
	department.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, department)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, fmt.Errorf("nama departemen sudah ada")
		}
		return nil, fmt.Errorf("gagal membuat departemen: %w", err)
	}
	return result, nil
}

func (r *departmentRepository) GetAllDepartments(ctx context.Context) ([]models.Department, error) {
	var departments []models.Department
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("gagal menemukan departemen: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &departments); err != nil {
		return nil, fmt.Errorf("gagal mendecode departemen: %w", err)
	}
	return departments, nil
}

func (r *departmentRepository) GetDepartmentByID(ctx context.Context, id primitive.ObjectID) (*models.Department, error) {
	var department models.Department
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&department)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("departemen tidak ditemukan")
		}
		return nil, fmt.Errorf("gagal menemukan departemen berdasarkan ID: %w", err)
	}
	return &department, nil
}

func (r *departmentRepository) UpdateDepartment(ctx context.Context, id primitive.ObjectID, updateData bson.M) (*mongo.UpdateResult, error) {
	updateData["updated_at"] = time.Now()
	filter := bson.M{"_id": id}
	update := bson.M{"$set": updateData}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("gagal mengupdate departemen: %w", err)
	}
	return result, nil
}

func (r *departmentRepository) DeleteDepartment(ctx context.Context, id primitive.ObjectID) (*mongo.DeleteResult, error) {
	filter := bson.M{"_id": id}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("gagal menghapus departemen: %w", err)
	}
	return result, nil
}

func (r *departmentRepository) FindDepartmentByName(ctx context.Context, name string) (*models.Department, error) {
	var department models.Department
	filter := bson.M{"name": name}
	err := r.collection.FindOne(ctx, filter).Decode(&department)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("departemen tidak ditemukan")
		}
		return nil, fmt.Errorf("gagal menemukan departemen berdasarkan nama: %w", err)
	}
	return &department, nil
}

// <--- BARU: Implementasi method CountDocuments untuk tipe *departmentRepository
func (r *departmentRepository) CountDocuments(ctx context.Context, filter bson.M) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung dokumen dari koleksi departemen: %w", err)
	}
	return count, nil
}