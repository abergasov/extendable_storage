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
		uuid.NewString(): testhelpers.GenerateMBData(t, 2.1),
		uuid.NewString(): testhelpers.GenerateMBData(t, 3.2),
	}
	container := testhelpers.GetClean(t)
	storageClusterUsage := map[string]float64{
		"NODE_A": 0,
		"NODE_B": 0,
		"NODE_C": 0,
		"NODE_D": 0,
		"NODE_E": 0,
		"NODE_F": 0,
	}
	storageClusters := make(map[string]storager.DataKeeper)
	for storageName := range storageClusterUsage {
		srv := addStorage(t, container, storageName, 6)
		storageClusters[storageName] = srv
	}
	container.ServiceOrchestrator.PrintServerPositions()

	// when
	for id, data := range dataMap {
		require.NoError(t, container.ServiceReceiver.SaveFile(container.Ctx, id, data))
	}
	t.Logf("data saved")

	// then
	for id, data := range dataMap {
		receivedData, err := container.ServiceReceiver.GetFile(container.Ctx, id)
		require.NoError(t, err)
		require.Equal(t, data, receivedData)
	}

	for storageName, srv := range storageClusters {
		usage, err := srv.GetUsage()
		require.NoError(t, err)
		storageClusterUsage[storageName] = usage
		t.Logf("%s usage: %.2f", storageName, usage)
	}

	t.Run("data should be rebalanced after new server added", func(t *testing.T) {
		// given
		srvG := addStorage(t, container, "NODE_G", 10)
		srvH := addStorage(t, container, "NODE_H", 10)
		container.ServiceOrchestrator.PrintServerPositions()

		// when
		for id, data := range dataMap {
			require.ErrorContains(t, container.ServiceReceiver.SaveFile(container.Ctx, id, data), "file already exists")
		}

		// then
		for id, data := range dataMap {
			receivedData, err := container.ServiceReceiver.GetFile(container.Ctx, id)
			if err != nil {
				container.ServiceOrchestrator.PrintServerPositions()
			}
			require.NoError(t, err)
			require.Equal(t, data, receivedData)
		}

		counter := 0
		for node, usage := range storageClusterUsage {
			newUsage, err := storageClusters[node].GetUsage()
			require.NoError(t, err)
			if newUsage != usage {
				counter++
			}
		}
		require.True(t, counter > 0)
		usageG, err := srvG.GetUsage()
		require.NoError(t, err)
		t.Logf("G usage: %.2f", usageG)
		usageH, err := srvH.GetUsage()
		require.NoError(t, err)
		t.Logf("H usage: %.2f", usageH)
	})
}

func addStorage(t *testing.T, container *testhelpers.TestContainer, name string, maxLimitMB int) storager.DataKeeper {
	srv := storager.NewService(container.Ctx, &storager.Config{
		MaxLimitMB: maxLimitMB,
		NodeID:     name,
		DataDir:    t.TempDir(),
	}, container.Logger)
	require.NoError(t, container.ServiceOrchestrator.AddDataKeeper(name, srv))
	return srv
}
