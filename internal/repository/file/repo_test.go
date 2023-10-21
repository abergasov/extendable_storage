package file_test

import (
	"extendable_storage/internal/entities"
	testhelpers "extendable_storage/internal/test_helpers"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRepo_FileChunksCRUD(t *testing.T) {
	// given
	container := testhelpers.GetClean(t)
	fileID := uuid.NewString()
	chunks := []*entities.FileChunk{
		{
			FileID:  uuid.NewString(),
			ChunkID: uuid.NewString(),
		},
		{
			FileID:  uuid.NewString(),
			ChunkID: uuid.NewString(),
		},
	}

	// when
	require.NoError(t, container.RepoFile.SaveFileChunks(container.Ctx, fileID, chunks))

	// then
	receivedChunks, err := container.RepoFile.GetFileChunks(container.Ctx, fileID)
	require.NoError(t, err)
	require.Equal(t, chunks, receivedChunks)

	files, err := container.RepoFile.GetChunksByStatus(container.Ctx, entities.FileStatusNew)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, fileID, files[0].ID)
	require.Equal(t, chunks, files[0].Chunks)

	t.Run("should serve by date and status", func(t *testing.T) {
		files, err = container.RepoFile.GetChunksUpdatedBeforeDataWithStatus(container.Ctx, entities.FileStatusNew, time.Now().Add(1*time.Hour))
		require.NoError(t, err)
		require.Len(t, files, 1)
	})

	t.Run("should delete file", func(t *testing.T) {
		// when
		require.NoError(t, container.RepoFile.DeleteFile(container.Ctx, fileID))

		// then
		files, err = container.RepoFile.GetChunksByStatus(container.Ctx, entities.FileStatusNew)
		require.NoError(t, err)
		require.Len(t, files, 0)
	})
}
