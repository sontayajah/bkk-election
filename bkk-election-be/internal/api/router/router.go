package router

import (
	"github.com/gin-gonic/gin"
	
	"github.com/sontayajah/bkk-election/bkk-election-be/internal/api/handler"
	"github.com/sontayajah/bkk-election/bkk-election-be/internal/queue"
)

// SetupRoutes ทำหน้าที่เตรียม Endpoint ทั้งหมดของแอปพลิเคชัน
func SetupRoutes(producer *queue.KafkaProducer) *gin.Engine {
	// สร้าง Gin Router
	r := gin.Default()

	// 1. สร้าง (Instantiate) Handlers ที่ต้องใช้งาน
	stationHandler := handler.NewStationHandler(producer)

	// 2. จัดกลุ่ม API Routes (เพื่อความสะอาดตา)
	api := r.Group("/api")
	{
		// Health Check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "UP", "kafka": "CONNECTED"})
		})

		// 🗳️ Election Endpoints
		stations := api.Group("/stations")
		{
			// ผูก POST /api/stations/submit เข้ากับฟังก์ชันใน Handler
			stations.POST("/submit", stationHandler.SubmitResult)
		}
	}

	return r
}