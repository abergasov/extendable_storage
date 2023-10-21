package testhelpers

import (
	"context"
	"extendable_storage/internal/config"
	"extendable_storage/internal/logger"
	"extendable_storage/internal/repository/file"
	"extendable_storage/internal/service/orchestrator"
	"extendable_storage/internal/service/receiver"
	"extendable_storage/internal/storage/database"
	"fmt"
	"os"
	"time"

	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type TestContainer struct {
	Ctx    context.Context
	Logger logger.AppLogger

	RepoFile *file.Repo

	ServiceOrchestrator orchestrator.DataRouter
	ServiceReceiver     receiver.DataReceiver
}

func GetClean(t *testing.T) *TestContainer {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conf := getTestConfig()
	prepareTestDB(t, &conf.ConfigDB)

	dbConnect, err := database.InitDBConnect(&conf.ConfigDB, guessMigrationDir(t))
	require.NoError(t, err)
	cleanupDB(t, dbConnect)
	t.Cleanup(func() {
		require.NoError(t, dbConnect.Client().Close())
	})

	appLog := logger.NewAppSLogger("test")
	// repo init
	repoFile := file.InitRepo(dbConnect)

	// service init
	serviceDataOrchestrator := orchestrator.NewService(ctx, appLog)
	serviceDataReceiver := receiver.NewService(ctx, appLog, serviceDataOrchestrator, repoFile)
	t.Cleanup(func() {
		cancel()
		serviceDataReceiver.Stop()
	})
	return &TestContainer{
		Ctx:    ctx,
		Logger: appLog,

		RepoFile: repoFile,

		ServiceOrchestrator: serviceDataOrchestrator,
		ServiceReceiver:     serviceDataReceiver,
	}
}

func prepareTestDB(t *testing.T, cnf *config.DBConf) {
	dbConnect, err := database.InitDBConnect(&config.DBConf{
		Address:        cnf.Address,
		Port:           cnf.Port,
		User:           cnf.User,
		Pass:           cnf.Pass,
		DBName:         "postgres",
		MaxConnections: cnf.MaxConnections,
	}, "")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, dbConnect.Client().Close())
	}()
	if _, err = dbConnect.Client().Exec(fmt.Sprintf("CREATE DATABASE %s", cnf.DBName)); !isDatabaseExists(err) {
		require.NoError(t, err)
	}
}

func getTestConfig() *config.AppConfig {
	return &config.AppConfig{
		AppPort: 0,
		ConfigDB: config.DBConf{
			Address:        "localhost",
			Port:           "5449",
			User:           "aHAjeK",
			Pass:           "AOifjwelmc8dw",
			DBName:         "sybill_test",
			MaxConnections: 10,
		},
	}
}

func isDatabaseExists(err error) bool {
	return checkSQLError(err, "42P04")
}

func checkSQLError(err error, code string) bool {
	if err == nil {
		return false
	}
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return false
	}
	return string(pqErr.Code) == code
}

func guessMigrationDir(t *testing.T) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	res := strings.Split(dir, "/internal")
	return res[0] + "/migrations"
}

func cleanupDB(t *testing.T, connector database.DBConnector) {
	tables := []string{"files"}
	for _, table := range tables {
		_, err := connector.Client().Exec(fmt.Sprintf("TRUNCATE %s CASCADE", table))
		require.NoError(t, err)
	}
}
