package station

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sontayajah/bkk-election/bkk-election-be/internal/infra/queue"
)

// ResultPayload ย้าย Struct มาไว้ใน Domain ของมันเอง
type ResultPayload struct {
	IdempotencyKey string          `json:"idempotency_key"`
	DistrictID     int             `json:"district_id"`
	StationID      int             `json:"polling_station_id"`
	VotersCount    int             `json:"voters_count"`
	ValidBallots   int             `json:"valid_ballots"`
	InvalidBallots int             `json:"invalid_ballots"`
	NoVotes        int             `json:"no_votes"`
	CandidateVotes []CandidateVote `json:"candidate_votes"`
}

type CandidateVote struct {
	CandidateID int `json:"candidate_id"`
	Votes       int `json:"votes"`
}

// Handler จัดการ HTTP Request สำหรับโดเมนหน่วยเลือกตั้ง
type Handler struct {
	producer *queue.KafkaProducer
}

func NewHandler(p *queue.KafkaProducer) *Handler {
	return &Handler{producer: p}
}

// SubmitResult API สำหรับรับผลคะแนน
func (h *Handler) SubmitResult(c *gin.Context) {
	var payload ResultPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Data Validation
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

	// สร้าง Idempotency Key
	payload.IdempotencyKey = fmt.Sprintf("district_%d_station_%d", payload.DistrictID, payload.StationID)

	// แปลง Struct ของ Domain ให้เป็น Struct ของ Queue ก่อนส่ง (เพื่อลดการผูกมัด)
	queuePayload := queue.StationResultPayload{
		IdempotencyKey: payload.IdempotencyKey,
		DistrictID:     payload.DistrictID,
		StationID:      payload.StationID,
		VotersCount:    payload.VotersCount,
		ValidBallots:   payload.ValidBallots,
		InvalidBallots: payload.InvalidBallots,
		NoVotes:        payload.NoVotes,
	}
	for _, cv := range payload.CandidateVotes {
		queuePayload.CandidateVotes = append(queuePayload.CandidateVotes, queue.CandidateVote{
			CandidateID: cv.CandidateID,
			Votes:       cv.Votes,
		})
	}

	// โยนลง Kafka
	err := h.producer.PublishStationResult(c.Request.Context(), queuePayload)
	if err != nil {
		log.Printf("❌ Failed to publish to Kafka: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process submission"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "รับผลคะแนนเข้าสู่ระบบเรียบร้อยแล้ว กำลังประมวลผล",
		"tracking_id": payload.IdempotencyKey,
	})
}