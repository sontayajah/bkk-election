-- ==========================================
-- BKK Election - SQL Queries (For sqlc)
-- ==========================================

-- 1. บันทึกข้อมูลสรุปของหน่วยเลือกตั้ง
-- ใช้ RETURNING id เพื่อเอา id ไปผูกกับตารางคะแนนย่อย
-- name: CreateStationSubmission :one
INSERT INTO station_submissions (
    idempotency_key, polling_station_id, district_id, voters_count, valid_ballots, invalid_ballots, no_votes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING id;

-- 2. บันทึกคะแนนดิบของผู้สมัครในหน่วยนั้น
-- Go จะรันคำสั่งนี้วนลูปตามจำนวนผู้สมัครที่ส่งคะแนนมา
-- name: CreateStationCandidateVote :exec
INSERT INTO station_candidate_votes (
    submission_id, candidate_id, votes
) VALUES (
    $1, $2, $3
);

-- 3. อัปเดตคะแนนรวมของเขตแบบ Atomic
-- ถ้าไม่มีข้อมูลเขตนี้/เบอร์นี้ ให้ INSERT ใหม่ 
-- แต่ถ้ามีแล้ว (ON CONFLICT) ให้เอาคะแนนเดิม + คะแนนใหม่
-- name: UpsertDistrictCandidateSummary :exec
INSERT INTO district_summaries (
    district_id, candidate_id, total_votes
) VALUES (
    $1, $2, $3
) ON CONFLICT (district_id, candidate_id)
DO UPDATE SET
    total_votes = district_summaries.total_votes + EXCLUDED.total_votes,
    last_updated = CURRENT_TIMESTAMP;

-- 4. อัปเดตสถิติรวมของเขตแบบ Atomic
-- บวกยอดรวมบัตร และบวกจำนวนหน่วยที่ส่งแล้ว (submitted_stations + 1)
-- name: UpsertDistrictStats :exec
INSERT INTO district_stats (
    district_id, submitted_stations, total_voters_count, total_valid_ballots, total_invalid_ballots, total_no_votes
) VALUES (
    $1, 1, $2, $3, $4, $5
) ON CONFLICT (district_id)
DO UPDATE SET
    submitted_stations = district_stats.submitted_stations + 1,
    total_voters_count = district_stats.total_voters_count + EXCLUDED.total_voters_count,
    total_valid_ballots = district_stats.total_valid_ballots + EXCLUDED.total_valid_ballots,
    total_invalid_ballots = district_stats.total_invalid_ballots + EXCLUDED.total_invalid_ballots,
    total_no_votes = district_stats.total_no_votes + EXCLUDED.total_no_votes,
    last_updated = CURRENT_TIMESTAMP;


-- ==========================================
-- Queries สำหรับดึงข้อมูล (API / Read Model)
-- ==========================================

-- ดึง Leaderboard 5 อันดับแรกของ กทม. (ใช้ SUM รวมทุกเขต)
-- name: GetOverallLeaderboard :many
SELECT candidate_id, SUM(total_votes)::bigint AS total_votes
FROM district_summaries
GROUP BY candidate_id
ORDER BY total_votes DESC
LIMIT 5;

-- ดึงข้อมูลสถิติของทุกเขต เพื่อเอาไปทำ Progress Bar 
-- name: GetAllDistrictStats :many
SELECT * FROM district_stats ORDER BY district_id;