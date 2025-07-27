package repository

import (
	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"
	util "Sistem-Manajemen-Karyawan/pkg/utils"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/teambition/rrule-go"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type WorkScheduleRepository struct {
	Collection *mongo.Collection
}

func (r *WorkScheduleRepository) FindUserScheduleForDate(ctx *fasthttp.RequestCtx, userID primitive.ObjectID, today string) (any, error) {
	panic("unimplemented")
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
            "start_time":      payload.StartTime,
            "end_time":        payload.EndTime,
            "note":            payload.Note,
            "recurrence_rule": payload.RecurrenceRule, 
            "updated_at":      time.Now(),
        },
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := r.Collection.UpdateByID(ctx, id, update) 
    if err != nil {
        return err
    }
    if result.MatchedCount == 0 {
        return errors.New("jadwal tidak ditemukan")
    }
    return nil
}

func (r *WorkScheduleRepository) FindByID(id primitive.ObjectID) (*models.WorkSchedule, error) {
	filter := bson.M{"_id": id} // Filter berdasarkan field _id

	var result models.WorkSchedule
	err := r.Collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("jadwal tidak ditemukan")
		}
		return nil, err
	}
	return &result, nil
}

func (r *WorkScheduleRepository) FindApplicableScheduleForUser(ctx context.Context, userID primitive.ObjectID, date string) (*models.WorkSchedule, error) {
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("format tanggal tidak valid: %s", date)
	}

	// --- LOG 1: Memulai proses pencarian ---
	log.Printf("[DEBUG] Mencari jadwal untuk User: %s pada Tanggal: %s", userID.Hex(), date)

	// Cek hari libur
	holidayMap, err := util.GetHolidayMap(targetDate.Format("2006"))
	if err != nil {
		log.Printf("[WARN] Gagal mengambil data hari libur: %v", err)
	}
	if holidayMap != nil && holidayMap[date] {
		log.Printf("[DEBUG] Tanggal %s adalah hari libur. Pencarian dihentikan.", date)
		return nil, errors.New("jadwal tidak ditemukan (hari libur)")
	}

	// Filter yang kuat untuk mengambil jadwal spesifik ATAU jadwal umum
	filter := bson.M{
		"$or": []bson.M{
			{"user_id": userID},
			{"user_id": nil},
			{"user_id": bson.M{"$exists": false}},
		},
	}

	applicableRules, err := r.FindAllWithFilter(filter)
	if err != nil {
		log.Printf("[ERROR] Gagal query ke database: %v", err)
		return nil, fmt.Errorf("gagal mengambil aturan jadwal: %w", err)
	}

	// --- LOG 2: Hasil query dari database ---
	log.Printf("[DEBUG] Ditemukan %d aturan yang mungkin berlaku untuk user/umum.", len(applicableRules))

	if len(applicableRules) == 0 {
		return nil, errors.New("jadwal tidak ditemukan")
	}

	// Variabel untuk menyimpan jadwal dengan prioritas
	var specificSchedule *models.WorkSchedule
	var globalSchedule *models.WorkSchedule

	for i := range applicableRules {
		rule := applicableRules[i]
		isApplicable := false
		instanceDate := date // Default ke tanggal yang dicari

		// Cek apakah aturan ini berlaku untuk tanggal yang dicari
		if rule.RecurrenceRule == "" {
			if rule.Date == date {
				isApplicable = true
			}
		} else {
			rOption, err := rrule.StrToROption(rule.RecurrenceRule)
			if err != nil { continue }
			
			ruleStartDate, _ := time.Parse("2006-01-02", rule.Date)
			rOption.Dtstart = ruleStartDate
			
			rr, err := rrule.NewRRule(*rOption)
			if err != nil { continue }
			
			if len(rr.Between(targetDate, targetDate, true)) > 0 {
				isApplicable = true
			}
		}

		if isApplicable {
			// --- LOG 3: Aturan yang cocok ditemukan ---
			log.Printf("[DEBUG]   -> Aturan ID %s COCOK untuk tanggal %s.", rule.ID.Hex(), date)

			instance := rule
			instance.Date = instanceDate

			// Cek prioritas: Spesifik > Umum
			if rule.UserID != nil && !rule.UserID.IsZero() {
				log.Printf("[DEBUG]     -> Ditemukan sebagai JADWAL SPESIFIK. Pencarian dihentikan.")
				specificSchedule = &instance
				break // Jadwal spesifik punya prioritas tertinggi, langsung hentikan loop
			} else {
				log.Printf("[DEBUG]     -> Ditemukan sebagai JADWAL UMUM.")
				globalSchedule = &instance
			}
		}
	}

	// Kembalikan hasil berdasarkan prioritas
	if specificSchedule != nil {
		log.Printf("[DEBUG] => FINAL: Mengembalikan JADWAL SPESIFIK untuk tanggal %s.", date)
		return specificSchedule, nil
	}
	if globalSchedule != nil {
		log.Printf("[DEBUG] => FINAL: Mengembalikan JADWAL UMUM untuk tanggal %s.", date)
		return globalSchedule, nil
	}

	log.Printf("[DEBUG] => FINAL: TIDAK ADA JADWAL yang cocok untuk tanggal %s setelah diproses.", date)
	return nil, errors.New("jadwal tidak ditemukan")
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
