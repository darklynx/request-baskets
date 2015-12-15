package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

var masterToken, _ = GenerateToken()

// StartServer starts RequestBasket server
func StartServer() {
	// TODO: implement support for server config
	log.Printf("Master token: %s", GetMasterToken())

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
	router.ServeFiles("/"+WEB_ROOT+"/*filepath", http.Dir("./src/github.com/darklynx/request-baskets/web"))

	// basket requests
	router.NotFound = http.HandlerFunc(AcceptBasketRequests)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", DEFAULT_PORT), router))
}

func GetMasterToken() string {
	return masterToken
}
