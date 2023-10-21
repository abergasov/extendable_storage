package service_test

import (
	"extendable_storage/internal/service/storager"
	testhelpers "extendable_storage/internal/test_helpers"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSaveData(t *testing.T) {
	// given
	dataMap := map[string][]byte{
		uuid.NewString(): testhelpers.GenerateMBData(t, 1),
		uuid.NewString(): testhelpers.GenerateMBData(t, 3.1),
		uuid.NewString(): testhelpers.GenerateMBData(t, 4.2),
	}
	container := testhelpers.GetClean(t)
	srvA := storager.NewService(container.Ctx, &storager.Config{
		MaxLimitMB: 10,
		NodeID:     "A",
		DataDir:    t.TempDir(),
	}, container.Logger)
	srvB := storager.NewService(container.Ctx, &storager.Config{
		MaxLimitMB: 7,
		NodeID:     "B",
		DataDir:    t.TempDir(),
	}, container.Logger)
	srvC := storager.NewService(container.Ctx, &storager.Config{
		MaxLimitMB: 15,
		NodeID:     "C",
		DataDir:    t.TempDir(),
	}, container.Logger)
	require.NoError(t, container.ServiceOrchestrator.AddDataKeeper("A", srvA))
	require.NoError(t, container.ServiceOrchestrator.AddDataKeeper("B", srvB))
	require.NoError(t, container.ServiceOrchestrator.AddDataKeeper("C", srvC))
	container.ServiceOrchestrator.MarkServerReady("A")
	container.ServiceOrchestrator.MarkServerReady("B")
	container.ServiceOrchestrator.MarkServerReady("C")

	// when
	for id, data := range dataMap {
		require.NoError(t, container.ServiceReceiver.SaveFile(container.Ctx, id, data))
	}

	// then
	for id, data := range dataMap {
		receivedData, err := container.ServiceReceiver.GetFile(container.Ctx, id)
		require.NoError(t, err)
		require.Equal(t, data, receivedData)
	}
	usageA, err := srvA.GetUsage()
	require.NoError(t, err)
	t.Logf("A usage: %d", usageA)
	usageB, err := srvB.GetUsage()
	require.NoError(t, err)
	t.Logf("B usage: %d", usageB)
	usageC, err := srvC.GetUsage()
	require.NoError(t, err)
	t.Logf("C usage: %d", usageC)
}
