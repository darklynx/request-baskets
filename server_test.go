package main

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testServer *http.Server

func TestMain(m *testing.M) {
	testsSetup()
	code := m.Run()
	testsShutdown()
	os.Exit(code)
}

func testsSetup() {
	// global config
	serverConfig = CreateConfig()
	// global server creation with default settings (performs some global initialization)
	testServer = CreateServer(serverConfig)
}

func testsShutdown() {
	// release global DB
	basketsDb.Release()
}

func TestCreateServer(t *testing.T) {
	assert.NotNil(t, testServer, "HTTP server is expected")
	assert.NotNil(t, basketsDb, "shared instance of basket database is expected")
	assert.NotNil(t, httpClient, "default HTTP client is expected")
	assert.NotNil(t, httpInsecureClient, "insecure HTTP client is expected")
}

func TestCreateServer_UnknownDbType(t *testing.T) {
	assert.Nil(t, CreateServer(&ServerConfig{DbType: "xyz"}), "Server is not expected")
}

func TestCreateBasketsDatabase(t *testing.T) {
	memdb := createBasketsDatabase(DbTypeMemory, "./mem", "")
	if assert.NotNil(t, memdb, "In-memory baskets database is expected") {
		memdb.Release()
	}

	boltfile := "./bolt_database.db"
	boltdb := createBasketsDatabase(DbTypeBolt, boltfile, "")
	if assert.NotNil(t, boltdb, "Bolt baskets database is expected") {
		boltdb.Release()
		os.Remove(boltfile)
	}

	sqldb := createBasketsDatabase(DbTypeSQL, pgTestConnection, "")
	if assert.NotNil(t, sqldb, "PostgreSQL database is expected") {
		sqldb.Release()
	}

	sqldbconn := createBasketsDatabase(DbTypeSQL, "./baskets.db", pgTestConnection)
	if assert.NotNil(t, sqldbconn, "PostgreSQL database is expected") {
		sqldbconn.Release()
	}

	assert.Nil(t, createBasketsDatabase("xyz", "./xyz", ""), "Database of unknown type is not expected")
}

func TestCreateDefaultBaskets(t *testing.T) {
	db := NewMemoryDatabase()
	defer db.Release()

	createDefaultBaskets(db, []string{"abc", "xyz", "illegal/name", "abc"})

	assert.Equal(t, 2, db.Size(), "wrong database size")
	assert.NotNil(t, db.Get("abc"), "default basket 'abc' is expected")
	assert.NotNil(t, db.Get("xyz"), "default basket 'xyz' is expected")

	assert.Equal(t, serverConfig.InitCapacity, db.Get("abc").Config().Capacity, "unexpected basket capacity")
}
