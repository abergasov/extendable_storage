package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"hash/crc32"
)

const (
	// ChunkSize 100KB
	ChunkSize     = 100 * 1024
	CircleSectors = 360
)

type FileChunk struct {
	// FileID is a unique identifier of the file. Based on user ID
	FileID string `json:"file_id"`
	// ChunkID hash of the chunk
	ChunkID string `json:"chunk_id"`
}

func (f *FileChunk) String() string {
	return f.FileID + "_" + f.ChunkID
}

func (f *FileChunk) Hash() uint32 {
	return crc32.ChecksumIEEE([]byte(f.String()))
}

func (f *FileChunk) Value() (driver.Value, error) {
	// Marshal the FileChunk struct to a JSON string.
	value, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (f *FileChunk) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	var data []byte
	switch src := src.(type) {
	case []byte:
		data = src
	case string:
		data = []byte(src)
	default:
		return fmt.Errorf("invalid data type for FileChunk: %T", src)
	}

	return json.Unmarshal(data, f)
}
