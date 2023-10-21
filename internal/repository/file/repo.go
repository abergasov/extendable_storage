package file

import (
	"context"
	"encoding/json"
	"extendable_storage/internal/entities"
	"extendable_storage/internal/storage/database"
	"time"
)

type Repo struct {
	db database.DBConnector
}

func InitRepo(db database.DBConnector) *Repo {
	return &Repo{db: db}
}

func (r *Repo) SaveFileChunks(ctx context.Context, fileID string, chunks []*entities.FileChunk) error {
	chunksJSON, err := json.Marshal(chunks)
	if err != nil {
		return err
	}
	toSave := entities.File{
		ID:         fileID,
		Status:     entities.FileStatusNew,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Chunks:     chunks,
		ChunksJSON: chunksJSON,
	}
	_, err = r.db.Client().ExecContext(ctx, `
		INSERT INTO files (id, status, created_at, updated_at, chunks)
		VALUES ($1, $2, $3, $4, $5)`,
		toSave.ID, toSave.Status, toSave.CreatedAt, toSave.UpdatedAt, toSave.ChunksJSON)
	return err
}

func (r *Repo) SetFileStatus(ctx context.Context, fileID string, status entities.FileStatus) error {
	_, err := r.db.Client().ExecContext(ctx, `UPDATE files SET status = $1, updated_at = NOW() WHERE id = $2`, status, fileID)
	return err
}

func (r *Repo) GetFileChunks(ctx context.Context, fileID string) ([]*entities.FileChunk, error) {
	var file entities.File
	if err := r.db.Client().QueryRowxContext(ctx, `SELECT * FROM files WHERE id = $1`, fileID).StructScan(&file); err != nil {
		return nil, err
	}
	var result []*entities.FileChunk
	if err := json.Unmarshal(file.ChunksJSON, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repo) GetChunksByStatus(ctx context.Context, status entities.FileStatus) ([]*entities.File, error) {
	return r.getFiles(ctx, `SELECT * FROM files WHERE status = $1`, status)
}

func (r *Repo) GetChunksUpdatedBeforeDataWithStatus(ctx context.Context, status entities.FileStatus, updatedBefore time.Time) ([]*entities.File, error) {
	return r.getFiles(ctx, `SELECT * FROM files WHERE status = $1 AND updated_at < $2`, status, updatedBefore)
}

func (r *Repo) getFiles(ctx context.Context, query string, params ...any) ([]*entities.File, error) {
	rows, err := r.db.Client().QueryxContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*entities.File
	for rows.Next() {
		var file entities.File
		if err = rows.StructScan(&file); err != nil {
			return nil, err
		}
		var chunks []*entities.FileChunk
		if err = json.Unmarshal(file.ChunksJSON, &chunks); err != nil {
			return nil, err
		}
		file.Chunks = chunks
		result = append(result, &file)
	}
	return result, nil
}

func (r *Repo) DeleteFile(ctx context.Context, fileID string) error {
	_, err := r.db.Client().ExecContext(ctx, `DELETE FROM files WHERE id = $1`, fileID)
	return err
}
