package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDefaultConfig(t *testing.T) {
	// serverConfig should be initialized by testsSetup function
	if assert.NotNil(t, serverConfig, "server configuration is expected") {
		assert.Equal(t, DEFAULT_DB_TYPE, serverConfig.DbType, "wrong db type")
		assert.Equal(t, DEFAULT_PORT, serverConfig.ServerPort, "wrong server port")
		assert.Equal(t, INIT_BASKET_CAPACITY, serverConfig.InitCapacity, "wrong initial capacity")
		assert.Equal(t, MAX_BASKET_CAPACITY, serverConfig.MaxCapacity, "wrong max capacity")
		assert.Equal(t, DEFAULT_PAGE_SIZE, serverConfig.PageSize, "wrong page size")
		assert.Equal(t, "./baskets.db", serverConfig.DbFile, "wrong DB file location")
		assert.NotEmpty(t, serverConfig.MasterToken, "expected randomly generated master token")
	}
}
