package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDefaultConfig(t *testing.T) {
	// serverConfig should be initialized by testsSetup function
	if assert.NotNil(t, serverConfig, "server configuration is expected") {
		assert.Equal(t, defaultDatabaseType, serverConfig.DbType, "wrong db type")
		assert.Equal(t, defaultServicePort, serverConfig.ServerPort, "wrong server port")
		assert.Equal(t, initBasketCapacity, serverConfig.InitCapacity, "wrong initial capacity")
		assert.Equal(t, maxBasketCapacity, serverConfig.MaxCapacity, "wrong max capacity")
		assert.Equal(t, defaultPageSize, serverConfig.PageSize, "wrong page size")
		assert.Equal(t, "./baskets.db", serverConfig.DbFile, "wrong DB file location")
		assert.NotEmpty(t, serverConfig.MasterToken, "expected randomly generated master token")
	}
}
