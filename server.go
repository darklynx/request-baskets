package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// StartServer starts RequestBasket server
func StartServer() {
	router := httprouter.New()

	router.GET("/baskets", GetBaskets)

	router.GET("/baskets/:basket", GetBasket)
	router.POST("/baskets/:basket", CreateBasket)
	router.PUT("/baskets/:basket", UpdateBasket)
	router.DELETE("/baskets/:basket", DeleteBasket)

	router.GET("/baskets/:basket/requests", GetBasketRequests)
	router.DELETE("/baskets/:basket/requests", ClearBasket)
	router.ServeFiles("/web/*filepath", http.Dir("./src/github.com/darklynx/request-basket/web"))
	router.NotFound = http.HandlerFunc(AcceptBasketRequests)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", DEFAULT_PORT), router))
}
