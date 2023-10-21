package entities

import "time"

type FileStatus string

const (
	FileStatusNew      FileStatus = "new"
	FileStatusComplete FileStatus = "complete"
	FileStatusPurge    FileStatus = "purge"
)

type File struct {
	ID         string       `json:"id" db:"id"`
	Status     FileStatus   `json:"status" db:"status"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at" db:"updated_at"`
	Chunks     []*FileChunk `json:"chunks" db:"-"`
	ChunksJSON []byte       `json:"-" db:"chunks"`
}
