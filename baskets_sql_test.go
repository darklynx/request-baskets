package main

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

// These are mostly error tests for SQL baskets database when the connection is wron or broken
// That is why this tests cannot be refred as tests related to the cirtain driver (PostgreSQL, MySQL, etc.)

func TestSQLDatabase_Create_InvalidDriver(t *testing.T) {
	assert.Nil(t, NewSQLDatabase("invalid_driver://user@localhost"), "expected to fail and return nil")
}

func TestSQLDatabase_Create_NoDriver(t *testing.T) {
	assert.Nil(t, NewSQLDatabase("user@localhost:8000"), "expected to fail and return nil")
}

func TestSQLDatabase_Create_FailedPing(t *testing.T) {
	assert.Nil(t, NewSQLDatabase("postgres://user@localhost:8907/baskets"), "expected to fail and return nil")
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
