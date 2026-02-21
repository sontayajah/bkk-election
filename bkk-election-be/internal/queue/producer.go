package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// StationResultPayload คือโครงสร้างข้อมูล (JSON) ที่เรารับมาจากหน้าเว็บ
// (ต้องตรงกับที่เราออกแบบไว้ใน PRD)
type StationResultPayload struct {
	IdempotencyKey string `json:"idempotency_key"` // 🚨 เราจะสร้างคีย์นี้ใน API ก่อนโยนลง Kafka
	DistrictID     int    `json:"district_id"`
	StationID      int    `json:"polling_station_id"`
	VotersCount    int    `json:"voters_count"`
	ValidBallots   int    `json:"valid_ballots"`
	InvalidBallots int    `json:"invalid_ballots"`
	NoVotes        int    `json:"no_votes"`
	CandidateVotes []CandidateVote `json:"candidate_votes"`
}

type CandidateVote struct {
	CandidateID int `json:"candidate_id"`
	Votes       int `json:"votes"`
}

// KafkaProducer เป็น Struct สำหรับจัดการการเชื่อมต่อ
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer สร้างการเชื่อมต่อไปยัง Kafka (เรียกใช้ตอนเปิดเซิร์ฟเวอร์)
func NewKafkaProducer(brokerAddress string, topic string) *KafkaProducer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{}, // กระจายโหลดให้เท่าๆ กัน
		// สำคัญมากสำหรับ High Concurrency: ไม่ต้องรอให้เขียนครบทุก Broker (Async)
		Async:    true, 
	}
	return &KafkaProducer{writer: w}
}

// PublishStationResult เป็นฟังก์ชันสำหรับโยน Payload ลง Kafka
func (p *KafkaProducer) PublishStationResult(ctx context.Context, payload StationResultPayload) error {
	// 1. แปลง Go Struct (Payload) ให้กลายเป็นก้อน JSON (Bytes)
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 2. สร้างข้อความที่จะส่ง (กำหนด Key เป็นรหัสเขต เพื่อให้คะแนนเขตเดียวกันวิ่งไปเรียงคิวที่ Partition เดียวกัน)
	msg := kafka.Message{
		Key:   []byte(payload.IdempotencyKey), // ใช้คีย์กันซ้ำเป็น Key ของ Message
		Value: bytes,
		Time:  time.Now(),
	}

	// 3. ยิงเข้า Kafka
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("❌ Failed to write messages to Kafka: %v", err)
		return err
	}

	return nil
}

// Close ปิดการเชื่อมต่อเมื่อเลิกใช้งาน
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}