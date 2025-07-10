package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Perbarui interface LeaveRequestRepository
type LeaveRequestRepository interface {
	Create(req *models.LeaveRequest) (*mongo.InsertOneResult, error)
	// UBAH TIPE KEMBALIAN DI SINI!
	FindAll() ([]models.LeaveRequestWithUser, error) // <--- HARUSNYA models.LeaveRequestWithUser
	FindByID(id primitive.ObjectID) (*models.LeaveRequest, error)
	UpdateStatus(id primitive.ObjectID, status string, note string) (*mongo.UpdateResult, error)
	UpdateAttachmentURL(id primitive.ObjectID, fileURL string) (*mongo.UpdateResult, error)
	CountPendingRequests(ctx context.Context) (int64, error)
	FindByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.LeaveRequest, error)
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

// UBAH IMPLEMENTASI FindAll() UNTUK MELAKUKAN LOOKUP
func (r *leaveRequestRepository) FindAll() ([]models.LeaveRequestWithUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var requests []models.LeaveRequestWithUser
	
	pipeline := mongo.Pipeline{
		bson.D{{
			Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "users"},
				{Key: "localField", Value: "user_id"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "user_info"},
			},
		}},
		bson.D{{
			Key: "$unwind",
			Value: bson.D{
				{Key: "path", Value: "$user_info"},
				{Key: "preserveNullAndEmptyArrays", Value: false}, 
			},
		}},
		bson.D{{
			Key: "$project",
			Value: bson.D{
				{Key: "_id", Value: 1},
				{Key: "user_id", Value: 1},
				{Key: "start_date", Value: 1},
				{Key: "end_date", Value: 1},
				{Key: "reason", Value: 1},
				{Key: "status", Value: 1},
				{Key: "note", Value: 1},
				{Key: "request_type", Value: 1},
				{Key: "attachment_url", Value: 1},
				{Key: "created_at", Value: 1},
				{Key: "updated_at", Value: 1},
				{Key: "user_name", Value: "$user_info.name"},
				{Key: "user_email", Value: "$user_info.email"},
				{Key: "user_photo", Value: "$user_info.photo"},
			},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("gagal melakukan agregasi untuk pengajuan dengan detail user: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &requests); err != nil {
		return nil, fmt.Errorf("gagal mendecode pengajuan dengan detail user: %w", err)
	}
    
    // <--- BARIS LOG DEBUG BARU
    // Cetak hanya item pertama untuk menghindari log terlalu panjang jika banyak data
    if len(requests) > 0 {
        log.Printf("DEBUG: Hasil FindAll LeaveRequests (item pertama): %+v\n", requests[0])
    } else {
        log.Println("DEBUG: FindAll LeaveRequests mengembalikan data kosong.")
    }
    // --- AKHIR BARIS LOG DEBUG ---

	return requests, nil
}


func (r *leaveRequestRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.LeaveRequest, error) {
	var requests []models.LeaveRequest
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}) // Urutkan dari terbaru

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("gagal mencari pengajuan cuti berdasarkan user ID: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &requests); err != nil {
		return nil, fmt.Errorf("gagal decode hasil pengajuan cuti: %w", err)
	}
	
	if len(requests) == 0 {
		return []models.LeaveRequest{}, nil // Ini akan mengembalikan slice kosong, bukan nil
	}
	return requests, nil
}


func (r *leaveRequestRepository) FindByID(id primitive.ObjectID) (*models.LeaveRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request models.LeaveRequest
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal menemukan pengajuan berdasarkan ID: %w", err)
	}
	return &request, nil
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
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return nil, fmt.Errorf("gagal mengupdate status pengajuan: %w", err)
	}
	return result, nil
}

func (r *leaveRequestRepository) UpdateAttachmentURL(id primitive.ObjectID, fileURL string) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	update := bson.M{"$set": bson.M{"attachment_url": fileURL}}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return nil, fmt.Errorf("gagal mengupdate URL lampiran: %w", err)
	}
	return result, nil
}

func (r *leaveRequestRepository) CountPendingRequests(ctx context.Context) (int64, error) {
	filter := bson.M{"status": "pending"}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung pengajuan tertunda: %w", err)
	}
	return count, nil
}