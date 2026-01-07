-- Add title column to stt_requests table
ALTER TABLE stt_requests
ADD COLUMN IF NOT EXISTS title TEXT;

-- Add index for title search (optional, for future search functionality)
CREATE INDEX IF NOT EXISTS idx_stt_title 
ON stt_requests (title) 
WHERE title IS NOT NULL;

