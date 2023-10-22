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
	t.Run("data should be rebalanced after new server added", func(t *testing.T) {
		// given
		srvD := storager.NewService(container.Ctx, &storager.Config{
			MaxLimitMB: 15,
			NodeID:     "D",
			DataDir:    t.TempDir(),
		}, container.Logger)
		require.NoError(t, container.ServiceOrchestrator.AddDataKeeper("D", srvD))

		srvF := storager.NewService(container.Ctx, &storager.Config{
			MaxLimitMB: 10,
			NodeID:     "F",
			DataDir:    t.TempDir(),
		}, container.Logger)
		require.NoError(t, container.ServiceOrchestrator.AddDataKeeper("F", srvF))

		// when
		for id, data := range dataMap {
			require.ErrorContains(t, container.ServiceReceiver.SaveFile(container.Ctx, id, data), "file already exists")
		}

		// then
		for id, data := range dataMap {
			receivedData, err := container.ServiceReceiver.GetFile(container.Ctx, id)
			require.NoError(t, err)
			require.Equal(t, data, receivedData)
		}
		usageAAfter, err := srvA.GetUsage()
		require.NoError(t, err)
		t.Logf("A usage: %d", usageAAfter)
		usageBAfter, err := srvB.GetUsage()
		require.NoError(t, err)
		t.Logf("B usage: %d", usageBAfter)
		usageCAfter, err := srvC.GetUsage()
		require.NoError(t, err)
		t.Logf("C usage: %d", usageCAfter)
		require.True(t, usageAAfter < usageA || usageBAfter < usageB || usageCAfter < usageC)
		usageD, err := srvD.GetUsage()
		require.NoError(t, err)
		t.Logf("D usage: %d", usageD)
		usageF, err := srvD.GetUsage()
		require.NoError(t, err)
		t.Logf("F usage: %d", usageF)
	})
}
