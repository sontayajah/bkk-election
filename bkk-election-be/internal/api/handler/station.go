package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sontayajah/bkk-election/bkk-election-be/internal/queue"
)

// StationHandler คือตัวจับคู่ (Controller) สำหรับหน่วยเลือกตั้ง
type StationHandler struct {
	producer *queue.KafkaProducer
}

// NewStationHandler เป็น Constructor เพื่อฉีด Dependency เข้ามา
func NewStationHandler(p *queue.KafkaProducer) *StationHandler {
	return &StationHandler{producer: p}
}

// SubmitResult คือ Logic ที่ย้ายมาจาก main.go
func (h *StationHandler) SubmitResult(c *gin.Context) {
	var payload queue.StationResultPayload

	// 1. อ่าน JSON
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// 2. Data Validation
	totalCandidateVotes := 0
	for _, cv := range payload.CandidateVotes {
		totalCandidateVotes += cv.Votes
	}

	if totalCandidateVotes != payload.ValidBallots {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("บัตรดี (%d) ไม่ตรงกับผลรวมคะแนนผู้สมัคร (%d)", payload.ValidBallots, totalCandidateVotes),
		})
		return
	}

	if (payload.ValidBallots + payload.InvalidBallots + payload.NoVotes) != payload.VotersCount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ยอดผู้มาใช้สิทธิไม่ตรงกับผลรวม"})
		return
	}

	// 3. สร้าง Idempotency Key
	payload.IdempotencyKey = fmt.Sprintf("district_%d_station_%d", payload.DistrictID, payload.StationID)

	// 4. โยนลง Kafka ผ่าน Producer ที่ถูกฉีดเข้ามา
	err := h.producer.PublishStationResult(c.Request.Context(), payload)
	if err != nil {
		log.Printf("❌ Failed to publish to Kafka: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process submission"})
		return
	}

	// 5. ตอบกลับ HTTP 202
	c.JSON(http.StatusAccepted, gin.H{
		"message":     "รับผลคะแนนเข้าสู่ระบบเรียบร้อยแล้ว กำลังประมวลผล",
		"tracking_id": payload.IdempotencyKey,
	})
}