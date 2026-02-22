package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer เป็น Struct สำหรับจัดการการเชื่อมต่อ
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer ตอนนี้รับแค่ Broker Address ไม่ผูกมัดกับ Topic แล้ว!
func NewKafkaProducer(brokerAddress string) *KafkaProducer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		// 🚨 ลบบรรทัด Topic: topic ออกไปเลย
		Balancer: &kafka.LeastBytes{},
		Async:    true,
	}
	return &KafkaProducer{writer: w}
}

// PublishJSON 🌟 ฟังก์ชันกลางสุดเทพ ที่โดเมนไหนก็เรียกใช้ได้ แค่บอกชื่อ Topic
func (p *KafkaProducer) PublishJSON(ctx context.Context, topic string, key string, payload interface{}) error {
	// 1. แปลง Struct อะไรก็ได้ (interface{}) ให้เป็น JSON
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 2. สร้างข้อความ พร้อมระบุ Topic เป้าหมาย
	msg := kafka.Message{
		Topic: topic,       // 🎯 กำหนด Topic ตรงนี้แทน!
		Key:   []byte(key), // ใช้ Key เพื่อรับประกันลำดับข้อมูล (Ordering)
		Value: bytes,
		Time:  time.Now(),
	}

	// 3. ยิงเข้า Kafka
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("❌ Failed to write messages to Kafka topic '%s': %v", topic, err)
		return err
	}

	return nil
}

// Close ปิดการเชื่อมต่อ
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

// โครงสร้างข้อมูลที่ใช้ร่วมกัน
type StationResultPayload struct {
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