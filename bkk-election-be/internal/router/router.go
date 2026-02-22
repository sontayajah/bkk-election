package router

import (
	"github.com/gin-gonic/gin"
	
	"github.com/sontayajah/bkk-election/bkk-election-be/internal/modules/station"
	"github.com/sontayajah/bkk-election/bkk-election-be/internal/infra/queue"
)

// SetupRoutes ทำหน้าที่เตรียม Endpoint ทั้งหมดของแอปพลิเคชัน
func SetupRoutes(producer *queue.KafkaProducer) *gin.Engine {
	r := gin.Default()

	// 1. สร้าง Handler ของแต่ละ Vertical Slice (Domain)
	stationHandler := station.NewHandler(producer)
	// สมมติในอนาคตมีโดเมนอื่น: 
	// leaderboardHandler := leaderboard.NewHandler(db)

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "UP", "kafka": "CONNECTED"})
		})

		// 🗳️ Domain: Station (หน่วยเลือกตั้ง)
		stations := api.Group("/stations")
		{
			stations.POST("/submit", stationHandler.SubmitResult)
		}
		
		// 📊 Domain: Leaderboard (ตัวอย่างในอนาคต)
		// leaderboards := api.Group("/leaderboards") ...
	}

	return r
}