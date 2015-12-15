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

	router.GET("/baskets", GetBaskets)

	router.GET("/baskets/:basket", GetBasket)
	router.POST("/baskets/:basket", CreateBasket)
	router.PUT("/baskets/:basket", UpdateBasket)
	router.DELETE("/baskets/:basket", DeleteBasket)

	router.GET("/baskets/:basket/requests", GetBasketRequests)
	router.DELETE("/baskets/:basket/requests", ClearBasket)
	router.ServeFiles("/web/*filepath", http.Dir("./src/github.com/darklynx/request-baskets/web"))
	router.NotFound = http.HandlerFunc(AcceptBasketRequests)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", DEFAULT_PORT), router))
}

func GetMasterToken() string {
	return masterToken
}
