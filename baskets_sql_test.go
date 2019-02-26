package main

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

// These are mostly error tests for SQL baskets database when the connection is wrong or broken
// That is why this tests cannot be referred as tests related to the cirtain driver (PostgreSQL, MySQL, etc.)

func TestSQLDatabase_Create_InvalidDriver(t *testing.T) {
	assert.Nil(t, NewSQLDatabase("invalid_driver://user@localhost"), "expected to fail and return nil")
}

func TestSQLDatabase_Create_NoDriver(t *testing.T) {
	assert.Nil(t, NewSQLDatabase("user@localhost:8000"), "expected to fail and return nil")
}

func TestSQLDatabase_Create_FailedPing(t *testing.T) {
	assert.Nil(t, NewSQLDatabase("postgres://user@localhost:8907/baskets"), "expected to fail and return nil")
}

func TestSQLDatabase_Get_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basketsDb := sqlDatabase{db: sqldb, dbType: "postgres"}
	basket := basketsDb.Get("anybasket")
	assert.Nil(t, basket, "basket is not expected")
}

func TestSQLDatabase_Delete_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basketsDb := sqlDatabase{db: sqldb, dbType: "postgres"}
	basketsDb.Delete("anybasket")
	// TODO: find out how to capture the log output for validation
}

func TestSQLDatabase_GetNames_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basketsDb := sqlDatabase{db: sqldb, dbType: "postgres"}
	page := basketsDb.GetNames(10, 0)
	if assert.NotNil(t, page, "page object with names is expected") {
		assert.Equal(t, 0, page.Count)
		assert.False(t, page.HasMore)
		assert.Empty(t, page.Names)
	}
}

func TestSQLDatabase_FindNames_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basketsDb := sqlDatabase{db: sqldb, dbType: "postgres"}
	page := basketsDb.FindNames("a", 10, 0)
	if assert.NotNil(t, page, "page object with names is expected") {
		assert.False(t, page.HasMore)
		assert.Empty(t, page.Names)
	}
}

func TestSQLBasket_InvalidBasket(t *testing.T) {
	// Note: this test is using connection string for PostgreSQL from: baskets_sql_pg_test.go
	name := "test199"
	db := NewSQLDatabase(pgTestConnection)
	defer db.Release()

	db.Create(name, BasketConfig{Capacity: 20})
	defer db.Delete(name)
	basket := db.Get(name)

	sqldb, _ := sql.Open("postgres", pgTestConnection)
	defer sqldb.Close()

	// corrupted GET response
	sqldb.Exec("INSERT INTO rb_responses (basket_name, http_method, response) VALUES ($1, 'GET', '{ abc... <<<')", name)
	assert.Nil(t, basket.GetResponse("GET"))

	// corrupted request data
	sqldb.Exec("INSERT INTO rb_requests (basket_name, request) VALUES ($1, '.... <<< data - broken json')", name)
	assert.Equal(t, 1, basket.Size(), "wrong number of collected requests")
	page := basket.GetRequests(10, 0)
	assert.NotNil(t, page, "requests page is expected")
	assert.Equal(t, 1, page.Count, "wrong Count of requests in page")
	assert.Equal(t, 0, len(page.Requests), "wrong number of Requests in page")

	findPage := basket.FindRequests("", "any", 10, 0)
	assert.NotNil(t, findPage, "requests page is expected")
	assert.Equal(t, 0, len(findPage.Requests), "wrong number of Requests in page")
}

func TestSQLBasket_Update_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basket := sqlBasket{db: sqldb, dbType: "postgres", name: "anybasket"}
	basket.Update(BasketConfig{Capacity: 10})
	// TODO: find out how to capture the log output for validation
}

func TestSQLBasket_Authorize_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basket := sqlBasket{db: sqldb, dbType: "postgres", name: "anybasket"}
	assert.False(t, basket.Authorize("token"))
}

func TestSQLBasket_GetResponse_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basket := sqlBasket{db: sqldb, dbType: "postgres", name: "anybasket"}
	assert.Nil(t, basket.GetResponse("GET"))
}

func TestSQLBasket_Add_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basket := sqlBasket{db: sqldb, dbType: "postgres", name: "anybasket"}
	basket.Add(createTestPOSTRequest("http://localhost/anybasket", "Hellow world", "text/plain"))
	// TODO: find out how to capture the log output for validation
}

func TestSQLBasket_Clear_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basket := sqlBasket{db: sqldb, dbType: "postgres", name: "anybasket"}
	basket.Clear()
	// TODO: find out how to capture the log output for validation
}

func TestSQLBasket_GetRequests_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basket := sqlBasket{db: sqldb, dbType: "postgres", name: "anybasket"}
	page := basket.GetRequests(10, 0)
	if assert.NotNil(t, page, "page object with requests is expected") {
		assert.Equal(t, 0, page.Count)
		assert.Equal(t, 0, page.TotalCount)
		assert.False(t, page.HasMore)
		assert.Empty(t, page.Requests)
	}
}

func TestSQLBasket_FindRequests_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basket := sqlBasket{db: sqldb, dbType: "postgres", name: "anybasket"}
	page := basket.FindRequests("q", "any", 10, 0)
	if assert.NotNil(t, page, "page object with requests is expected") {
		assert.False(t, page.HasMore)
		assert.Empty(t, page.Requests)
	}
}

// SQL errors tests in private methods

func TestSQLDatabase_getTopBaskets_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	defer sqldb.Close()

	basketsDb := sqlDatabase{db: sqldb, dbType: "postgres"}
	assert.Empty(t, basketsDb.getTopBaskets("SELECT x FROM FROM", 10), "no baskets are expected")
}

func TestSQLDatabase_getInt_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	defer sqldb.Close()

	basketsDb := sqlDatabase{db: sqldb, dbType: "postgres"}
	assert.Equal(t, 12, basketsDb.getInt("SELECT count(x) FROM FROM", 12), "default value is expected if SQL error occurs")
}

func TestSQLBasket_getInt_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	defer sqldb.Close()

	basket := sqlBasket{db: sqldb, dbType: "postgres", name: "anybasket"}
	assert.Equal(t, 42, basket.getInt("SELECT count(x) FROM FROM", 42), "default value is expected if SQL error occurs")
}

func TestSQLBasket_applyLimit_SQLError(t *testing.T) {
	sqldb, _ := sql.Open("postgres", pgTestConnection)
	sqldb.Close()

	basket := sqlBasket{db: sqldb, dbType: "postgres", name: "anybasket"}
	basket.applyLimit(-1)
	// TODO: find out how to capture the log output for validation
}
