package station

import (
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