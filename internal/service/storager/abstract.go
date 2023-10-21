package storager

import "extendable_storage/internal/entities"

// DataKeeper is an interface for data storage nodes which can join the cluster and store data at any time
type DataKeeper interface {
	// GetUsage returns the amount of data stored in percentage of usage. 0 - 100
	// this metric is used to determine the most used node in the cluster.
	// New node candidate will be added to the cluster to balance the usage.
	GetUsage() (int64, error)

	// GetFile returns a file by its ID and hash. ID is user defined, hash is calculated by the system
	GetFile(file entities.FileChunk) ([]byte, error)
	// SaveFile saves a file by its ID and hash. ID is user defined, hash is calculated by the system
	SaveFile(file entities.FileChunk, data []byte) error

	// SaveFromSource command to load batch of data from external source.
	SaveFromSource(chunks []entities.FileChunk, source string) error
	// CheckFilesExistence returns a map of fileIDs and their existence in the storage
	CheckFilesExistence(chunks []entities.FileChunk) (map[string]bool, error)
}