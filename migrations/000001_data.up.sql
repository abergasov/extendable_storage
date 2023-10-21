CREATE TABLE files (
   id VARCHAR(255) PRIMARY KEY,
   status VARCHAR(255),
   created_at TIMESTAMPTZ,
   updated_at TIMESTAMPTZ,
   chunks JSONB
);

-- Create an index on the "updated_at" column
CREATE INDEX idx_updated_at ON files(updated_at);

-- Create an index on the "status" column
CREATE INDEX idx_status ON files(status);