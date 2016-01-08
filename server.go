package main

import (
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

	// configure service HTTP router
	router := httprouter.New()

	// basket names
	router.GET("/"+BASKETS_ROOT, GetBaskets)

	// basket management
	router.GET("/"+BASKETS_ROOT+"/:basket", GetBasket)
	router.POST("/"+BASKETS_ROOT+"/:basket", CreateBasket)
	router.PUT("/"+BASKETS_ROOT+"/:basket", UpdateBasket)
	router.DELETE("/"+BASKETS_ROOT+"/:basket", DeleteBasket)

	// requests management
	router.GET("/"+BASKETS_ROOT+"/:basket/requests", GetBasketRequests)
	router.DELETE("/"+BASKETS_ROOT+"/:basket/requests", ClearBasket)

	// web pages
	router.GET("/", ForwardToWeb)
	router.GET("/"+WEB_ROOT, WebIndexPage)
	router.GET("/"+WEB_ROOT+"/:basket", WebBasketPage)
	//router.ServeFiles("/"+WEB_ROOT+"/*filepath", http.Dir("./src/github.com/darklynx/request-baskets/web"))

	// basket requests
	router.NotFound = http.HandlerFunc(AcceptBasketRequests)

	go shutdownHook()

	log.Printf("[info] starting HTTP server on port: %d", serverConfig.ServerPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverConfig.ServerPort), router))
}

func createBasketsDatabase() BasketsDatabase {
	switch serverConfig.DbType {
	case DB_TYPE_MEM:
		return NewMemoryDatabase()
	case DB_TYPE_BOLT:
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
