package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/sontayajah/bkk-election/bkk-election-be/internal/api/router"
	"github.com/sontayajah/bkk-election/bkk-election-be/internal/db"
	"github.com/sontayajah/bkk-election/bkk-election-be/internal/queue"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found. Falling back to system environment variables.")
	}

	// --- 1. Database Setup ---
	dbUrl := os.Getenv("BACKEND_DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("❌ BACKEND_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatalf("❌ Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// เช็คว่าต่อติดจริงๆ ไหม (Ping)
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("❌ Database is not responding: %v\n", err)
	}
	log.Println("✅ Successfully connected to PostgreSQL (Connection Pool Ready)!")

	queries := db.New(pool)
	_ = queries // เราจะส่ง queries ไปให้ Worker ใช้ในอนาคต

	// --- 2. Kafka Setup ---
	// ชี้ไปที่ Kafka Broker ใน Docker Compose ของเรา (พอร์ต 9092)
	kafkaBroker := os.Getenv("BACKEND_KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}
	kafkaTopic := "station-results" // ชื่อ Topic ที่เราจะโยนของลงไป

	producer := queue.NewKafkaProducer(kafkaBroker, kafkaTopic)
	defer producer.Close()
	log.Println("✅ Connected to Kafka Producer!")

	// --- 3. API Router Setup ---
	r := router.SetupRoutes(producer)

	// --- 4. Start Server ---
	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8081"
	}
	
	log.Printf("🔥 Starting server on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}