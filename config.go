package main

import (
	"flag"
	"fmt"
	"log"
)

const (
	DEFAULT_PORT         = 55555
	DEFAULT_PAGE_SIZE    = 20
	INIT_BASKET_CAPACITY = 200
	MAX_BASKET_CAPACITY  = 2000
	DEFAULT_DB_TYPE      = DB_TYPE_MEM
	BASKETS_ROOT         = "baskets"
	WEB_ROOT             = "web"
	BASKET_NAME          = `^[\w\d\-_\.]+$`
)

type ServerConfig struct {
	ServerPort   int
	InitCapacity int
	MaxCapacity  int
	PageSize     int
	MasterToken  string
	DbType       string
	DbFile       string
}

// CreateConfig creates server configuration base on application command line arguments
func CreateConfig() *ServerConfig {
	var port = flag.Int("p", DEFAULT_PORT, "HTTP service port")
	var initCapacity = flag.Int("size", INIT_BASKET_CAPACITY, "Initial basket size (capacity)")
	var maxCapacity = flag.Int("maxsize", MAX_BASKET_CAPACITY, "Maximum allowed basket size (max capacity)")
	var pageSize = flag.Int("page", DEFAULT_PAGE_SIZE, "Default page size")
	var masterToken = flag.String("token", "", "Master token, random token is generated if not provided")
	var dbType = flag.String("db", DEFAULT_DB_TYPE, fmt.Sprintf(
		"Baskets storage type: %s - in-memory, %s - Bolt DB", DB_TYPE_MEM, DB_TYPE_BOLT))
	var dbFile = flag.String("file", "./baskets.db", "Database location, only applicable for file databases")
	flag.Parse()

	var token = *masterToken
	if len(token) == 0 {
		token, _ = GenerateToken()
		log.Printf("[info] generated master token: %s", token)
	}

	return &ServerConfig{
		ServerPort:   *port,
		InitCapacity: *initCapacity,
		MaxCapacity:  *maxCapacity,
		PageSize:     *pageSize,
		MasterToken:  token,
		DbType:       *dbType,
		DbFile:       *dbFile}
}
