package main

import (
	"flag"
	"fmt"
	"log"
)

const (
	defaultServicePort  = 55555
	defaultPageSize     = 20
	initBasketCapacity  = 200
	maxBasketCapacity   = 2000
	defaultDatabaseType = DbTypeMemory
	serviceAPIPath      = "baskets"
	serviceUIPath       = "web"
	basketNamePattern   = `^[\w\d\-_\.]+$`
)

// ServerConfig describes server configuration.
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
	var port = flag.Int("p", defaultServicePort, "HTTP service port")
	var initCapacity = flag.Int("size", initBasketCapacity, "Initial basket size (capacity)")
	var maxCapacity = flag.Int("maxsize", maxBasketCapacity, "Maximum allowed basket size (max capacity)")
	var pageSize = flag.Int("page", defaultPageSize, "Default page size")
	var masterToken = flag.String("token", "", "Master token, random token is generated if not provided")
	var dbType = flag.String("db", defaultDatabaseType, fmt.Sprintf(
		"Baskets storage type: %s - in-memory, %s - Bolt DB", DbTypeMemory, DbTypeBolt))
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
