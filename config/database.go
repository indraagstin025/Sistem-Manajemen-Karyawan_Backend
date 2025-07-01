package config

import (
	"context"
	"os"
	"time"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var MongoConn *mongo.Client

var DBName string = "manajemen-karyawan-db"
var UserCollection string = "users"
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