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

var serverConfig *ServerConfig
var basketsDb BasketsDatabase
var httpClient *http.Client
var httpInsecureClient *http.Client

// StartServer starts Request Baskets server
func StartServer() {
	// read config
	serverConfig = CreateConfig()
	// create database
	basketsDb = createBasketsDatabase()
	if basketsDb == nil {
		log.Print("[error] failed to create basket database")
		return
	}

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
	//router.ServeFiles("/"+WEB_ROOT+"/*filepath", http.Dir("./src/github.com/darklynx/request-baskets/web"))

	// basket requests
	router.NotFound = http.HandlerFunc(AcceptBasketRequests)

	go shutdownHook()

	log.Printf("[info] starting HTTP server on port: %d", serverConfig.ServerPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverConfig.ServerPort), router))
}

func createBasketsDatabase() BasketsDatabase {
	switch serverConfig.DbType {
	case DbTypeMemory:
		return NewMemoryDatabase()
	case DbTypeBolt:
		return NewBoltDatabase(serverConfig.DbFile)
	default:
		log.Printf("[error] unknown database type: %s", serverConfig.DbType)
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
