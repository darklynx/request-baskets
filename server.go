package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/julienschmidt/httprouter"
)

var basketsDb BasketsDatabase
var httpClient *http.Client
var httpInsecureClient *http.Client

// CreateServer creates an instance of Request Baskets server
func CreateServer(config *ServerConfig) *http.Server {
	// create database
	db := createBasketsDatabase(config.DbType, config.DbFile, config.DbConnection)
	if db == nil {
		log.Print("[error] failed to create basket database")
		return nil
	}
	basketsDb = db

	// HTTP clients
	httpClient = new(http.Client)
	insecureTransport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	httpInsecureClient = &http.Client{Transport: insecureTransport}

	// configure service HTTP router
	router := httprouter.New()

	// basket names
	router.GET("/"+serviceAPIPath, GetBaskets)

	// basket management
	router.GET("/"+serviceAPIPath+"/:basket", GetBasket)
	router.POST("/"+serviceAPIPath+"/:basket", CreateBasket)
	router.PUT("/"+serviceAPIPath+"/:basket", UpdateBasket)
	router.DELETE("/"+serviceAPIPath+"/:basket", DeleteBasket)

	router.GET("/"+serviceAPIPath+"/:basket/responses/:method", GetBasketResponse)
	router.PUT("/"+serviceAPIPath+"/:basket/responses/:method", UpdateBasketResponse)

	// requests management
	router.GET("/"+serviceAPIPath+"/:basket/requests", GetBasketRequests)
	router.DELETE("/"+serviceAPIPath+"/:basket/requests", ClearBasket)

	// web pages
	router.GET("/", ForwardToWeb)
	router.GET("/"+serviceUIPath, WebIndexPage)
	router.GET("/"+serviceUIPath+"/:basket", WebBasketPage)
	//router.ServeFiles("/"+serviceUIPath+"/*filepath", http.Dir("./src/github.com/darklynx/request-baskets/web"))

	// basket requests
	router.NotFound = http.HandlerFunc(AcceptBasketRequests)

	log.Printf("[info] HTTP server is listening on %s:%d", serverConfig.ServerAddr, serverConfig.ServerPort)
	server := &http.Server{Addr: fmt.Sprintf("%s:%d", serverConfig.ServerAddr, serverConfig.ServerPort), Handler: router}

	go shutdownHook()
	return server
}

func createBasketsDatabase(dbtype string, file string, conn string) BasketsDatabase {
	switch dbtype {
	case DbTypeMemory:
		return NewMemoryDatabase()
	case DbTypeBolt:
		return NewBoltDatabase(file)
	case DbTypeSQL:
		if len(conn) > 0 {
			return NewSQLDatabase(conn)
		}
		return NewSQLDatabase(file)
	default:
		log.Printf("[error] unknown database type: %s", dbtype)
		return nil
	}
}

func shutdownHook() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("[info] received signal: %s, shutting down database", sig)
		basketsDb.Release()
		done <- true
	}()

	<-done
	log.Printf("[info] terminating server")
	os.Exit(0)
}

func getHTTPClient(insecure bool) *http.Client {
	if insecure {
		return httpInsecureClient
	}
	return httpClient
}
