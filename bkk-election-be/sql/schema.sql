-- ==========================================
-- BKK Election - Database Schema (PostgreSQL)
-- ==========================================

-- 1. Master Data: ตารางพรรคการเมือง
CREATE TABLE parties (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    color_hex VARCHAR(7) NOT NULL, -- สีประจำพรรคสำหรับแสดงบน UI แผนที่
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Master Data: ตารางผู้สมัครผู้ว่าฯ
CREATE TABLE candidates (
    id INT PRIMARY KEY,
    number INT NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    party_id INT REFERENCES parties(id),
    image_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Master Data: ตารางเขตเลือกตั้ง (50 เขต กทม.)
CREATE TABLE districts (
    id INT PRIMARY KEY, -- รหัสเขตอย่างเป็นทางการ
    name_th VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL
);

-- 3.5 Master Data: ตารางหน่วยเลือกตั้งย่อยในแต่ละเขต
CREATE TABLE polling_stations (
    id SERIAL PRIMARY KEY,
    district_id INT REFERENCES districts(id) NOT NULL,
    station_number INT NOT NULL, -- หน่วยเลือกตั้งที่ X
    location_name VARCHAR(255),
    eligible_voters INT DEFAULT 0, -- จำนวนผู้มีสิทธิเลือกตั้งในหน่วยนี้
    UNIQUE(district_id, station_number)
);

-- ==========================================
-- CQRS: The Write Model (Event Log)
-- ==========================================
-- 4. ตารางบันทึกการส่งผลคะแนนของหน่วยเลือกตั้ง (1 หน่วย ส่งได้ 1 ครั้ง)
CREATE TABLE station_submissions (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL, -- ป้องกัน Network Retry ซ้ำ
    polling_station_id INT REFERENCES polling_stations(id) UNIQUE NOT NULL, -- 🚨 บังคับ 1 หน่วย ส่งผลได้แค่ 1 ครั้งเท่านั้น
    district_id INT REFERENCES districts(id) NOT NULL,
    voters_count INT NOT NULL, -- ผู้มาใช้สิทธิ
    valid_ballots INT NOT NULL, -- บัตรดี
    invalid_ballots INT NOT NULL, -- บัตรเสีย
    no_votes INT NOT NULL, -- ไม่ออกเสียง
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4.1 ตารางบันทึกคะแนนดิบของผู้สมัครในแต่ละหน่วย
CREATE TABLE station_candidate_votes (
    id BIGSERIAL PRIMARY KEY,
    submission_id BIGINT REFERENCES station_submissions(id) ON DELETE CASCADE,
    candidate_id INT REFERENCES candidates(id) NOT NULL,
    votes INT NOT NULL,
    UNIQUE(submission_id, candidate_id)
);

-- สร้าง Index เพื่อความเร็วในการค้นหาประวัติย้อนหลัง
CREATE INDEX idx_station_submissions_created_at ON station_submissions(created_at);

-- ==========================================
-- CQRS: The Read Model (Pre-aggregated)
-- ==========================================
-- 5. ตารางสรุปผลคะแนนรวมรายเขต (ใช้อ่านและซิงค์ขึ้น Redis)
CREATE TABLE district_summaries (
    id BIGSERIAL PRIMARY KEY,
    district_id INT REFERENCES districts(id) NOT NULL,
    candidate_id INT REFERENCES candidates(id) NOT NULL,
    total_votes BIGINT NOT NULL DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 🚨 Constraint สำคัญที่ทำให้เราใช้คำสั่ง ON CONFLICT (Upsert) ได้
    UNIQUE(district_id, candidate_id) 
);

-- สร้าง Index เรียงลำดับคะแนนจากมากไปน้อย เพื่อให้ดึง Leaderboard รายเขตได้เร็วสุดๆ
CREATE INDEX idx_district_summaries_ranking ON district_summaries(district_id, total_votes DESC);

-- ==========================================
-- 5.1 ตารางสถิติรวมของเขต (ภาพรวมบัตรดี/เสีย และความคืบหน้า)
-- ==========================================
CREATE TABLE district_stats (
    district_id INT PRIMARY KEY REFERENCES districts(id),
    total_stations INT NOT NULL DEFAULT 0, -- จำนวนหน่วยทั้งหมดในเขตนี้
    submitted_stations INT NOT NULL DEFAULT 0, -- จำนวนหน่วยที่ส่งคะแนนแล้ว (ความคืบหน้าการนับคะแนน)
    total_voters_count INT NOT NULL DEFAULT 0,
    total_valid_ballots INT NOT NULL DEFAULT 0,
    total_invalid_ballots INT NOT NULL DEFAULT 0,
    total_no_votes INT NOT NULL DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);