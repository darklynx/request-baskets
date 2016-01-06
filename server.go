package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

var serverConfig *ServerConfig
var basketsDb BasketsDatabase

// StartServer starts Request Baskets server
func StartServer() {
	// read config
	serverConfig = CreateConfig()
	// create database
	basketsDb = NewMemoryDatabase()

	//botl := NewBoltDatabase("requests.db")
	//botl.Release()

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

	log.Printf("Starting HTTP server on port: %d", serverConfig.ServerPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverConfig.ServerPort), router))
}
