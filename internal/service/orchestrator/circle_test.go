package orchestrator_test

import (
	"extendable_storage/internal/entities"
	"extendable_storage/internal/service/orchestrator"
	"extendable_storage/internal/service/storager"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCircle_AddServer(t *testing.T) {
	// given
	circle := orchestrator.NewCircle()

	// when
	mck := gomock.NewController(t)
	keys := map[string]struct{}{"A": {}, "B": {}, "C": {}, "D": {}, "E": {}, "F": {}, "G": {}, "H": {}, "I": {}, "J": {}}
	for key := range keys {
		srv := storager.NewMockDataKeeper(mck)
		srv.EXPECT().GetUsage().Return(uint64(randUsage()), nil).AnyTimes()
		_, _, _, err := circle.AddServer(fmt.Sprintf("srv%s", key), srv)
		require.NoError(t, err)
		circle.MarkServerReady(fmt.Sprintf("srv%s", key))
	}

	// then
	circle.PrintServerPositions()
	t.Run("should find server for data", func(t *testing.T) {
		// when, then
		for i := 0; i < 100_000; i++ {
			srv, srvName, err := circle.GetServerForChunk(&entities.FileChunk{
				FileID:  uuid.NewString(),
				ChunkID: uuid.NewString(),
			})
			require.NoError(t, err)
			require.NotNil(t, srv)
			_, ok := keys[strings.ReplaceAll(srvName, "srv", "")]
			require.True(t, ok)
		}
	})
}

func randUsage() int {
	minNum := 10
	maxNum := 100
	return minNum + rand.Intn(maxNum-minNum+1)
}
