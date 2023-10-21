package orchestrator

import "extendable_storage/internal/entities"

// DataRouter is an interface for data routing nodes which can join the cluster and route data at any time
// manage new nodes joining the cluster and route data to them
type DataRouter interface {
	// GetFileChunk returns a part of the file by its ID
	GetFileChunk(chunk *entities.FileChunk) ([]byte, error)
	// SaveFileChunk saves a part of the file by its ID
	SaveFileChunk(chunk *entities.FileChunk, data []byte) error

	// PurgeFileChunks command to purge file chunks
	PurgeFileChunks(chunks []*entities.FileChunk) error
}
