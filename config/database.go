package config

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var MongoConn *mongo.Client

// Nama Database
var DBName string = "manajemen-karyawan-db"

// Deklarasi semua nama koleksi di sini
var UserCollection string = "users"
var DepartmentCollection string = "departments"
var AttendanceCollection string = "attendances"
var SalaryCollection string = "salaries"
var LeaveRequestCollection string = "leave_requests"
var QRCodeCollection string = "qr_codes"

func MongoConnect() {
	mongoURI := os.Getenv("MONGOSTRING")

	if mongoURI == "" {
		log.Fatal("MONGOSTRING belum di setting di env. coba setting dulu")
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to create MongoDB client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	log.Println("Connected to MongoDB!")
	MongoConn = client
}

func InitDatabase() {
	if MongoConn == nil {
		log.Fatal("MongoDB client tidak diinisialisasi untuk InitDatabase. Panggil MongoConnect() terlebih dahulu.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userCollection := MongoConn.Database(DBName).Collection(UserCollection) // Menggunakan UserCollection dari file ini

	// Buat indeks unik untuk field 'email' di koleksi 'users'
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}}, // Indeks di field 'email', ascending
		Options: options.Index().SetUnique(true),  // Jadikan indeks unik
	}

	_, err := userCollection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Peringatan: Gagal membuat indeks unik untuk email di koleksi users: %v. Mungkin ada dokumen duplikat yang perlu dihapus manual.\n", err)
	} else {
		log.Println("Indeks unik untuk email berhasil dibuat di koleksi users.")
	}

	// Anda bisa menambahkan inisialisasi indeks lain di sini jika diperlukan
	// Contoh untuk departemen:
	// departmentCollection := MongoConn.Database(DBName).Collection(DepartmentCollection)
	// // Buat indeks unik untuk nama departemen jika diperlukan
	// indexDeptModel := mongo.IndexModel{
	// 	Keys:    bson.D{{Key: "name", Value: 1}},
	// 	Options: options.Index().SetUnique(true),
	// }
	// _, err = departmentCollection.Indexes().CreateOne(ctx, indexDeptModel)
	// if err != nil {
	// 	log.Printf("Peringatan: Gagal membuat indeks unik untuk nama departemen: %v\n", err)
	// } else {
	// 	log.Println("Indeks unik untuk nama departemen berhasil dibuat.")
	// }
}

func GetCollection(collectionName string) *mongo.Collection {
	if MongoConn == nil {
		log.Fatal("MongoDB untuk client tidak di inisialisasi. Panggil MongoConnect() first")
	}
	return MongoConn.Database(DBName).Collection(collectionName)
}

func DisconnectDB() {
	if MongoConn != nil {
		if err := MongoConn.Disconnect(context.Background()); err != nil {
			log.Fatalf("Error disconnecting from MongoDB: %v", err)
		}
		log.Println("Disconnect from MongoDB")
	}
}
