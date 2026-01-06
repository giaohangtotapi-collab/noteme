-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Main table: stt_requests
CREATE TABLE stt_requests (
  -- Primary key
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- User info
  user_id UUID NOT NULL,

  -- Audio info
  audio_url TEXT NOT NULL,
  audio_format TEXT,              -- m4a / wav / mp3
  audio_duration_ms INT,           -- duration in milliseconds
  audio_size_bytes INT,            -- optional, dùng để estimate cost

  -- STT info
  stt_provider TEXT NOT NULL,      -- google / fpt
  language TEXT DEFAULT 'vi-VN',
  model_version TEXT,              -- future-proof

  -- Result
  transcript TEXT,
  confidence REAL,                 -- nullable (Google STT không luôn trả)
  
  -- Status
  status TEXT NOT NULL,            -- processing / success / failed
  error_message TEXT,

  -- Performance
  processing_time_ms INT,

  -- Flexible metadata
  metadata JSONB DEFAULT '{}'::jsonb,

  -- Timestamps
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Indexes for performance
-- Lấy lịch sử theo user (use case chính)
CREATE INDEX idx_stt_user_created
ON stt_requests (user_id, created_at DESC);

-- Filter theo trạng thái
CREATE INDEX idx_stt_status
ON stt_requests (status);

-- Filter / analytics theo STT provider
CREATE INDEX idx_stt_provider
ON stt_requests (stt_provider);

