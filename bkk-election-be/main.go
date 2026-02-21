package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	// 🚨 เปลี่ยนตรงนี้ให้ตรงกับชื่อในบรรทัดแรกของไฟล์ bkk-election-be/go.mod ของคุณ
	"github.com/sontayajah/bkk-election/bkk-election-be/internal/db"
)

func main() {
	// 1. โหลดตัวแปรสภาพแวดล้อม (Environment Variables) จากไฟล์ .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found. Falling back to system environment variables.")
	}

	// 2. สร้าง Database Connection Pool (เตรียมรับมือ High Concurrency)
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

	// 3. ผูก Connection Pool เข้ากับคำสั่ง SQL ที่ sqlc สร้างให้เรา
	queries := db.New(pool)
	_ = queries // เดี๋ยวเราจะเอาตัวแปรนี้ไปส่งต่อให้ API และ Worker ใช้

	// 4. ตั้งค่า Gin Web Framework
	// ถ้าเป็น Production เราจะเซ็ต gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 5. สร้าง Route เริ่มต้น (Health Check)
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "UP",
			"database": "CONNECTED",
			"message": "BKK Election API is running 🚀",
		})
	})

	// 6. เริ่มเปิดเซิร์ฟเวอร์
	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("🔥 Starting server on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}