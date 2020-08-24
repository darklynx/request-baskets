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
var version *Version

// CreateServer creates an instance of Request Baskets server
func CreateServer(config *ServerConfig) *http.Server {
	version = &Version{
		Name:        serviceName,
		Version:     GitVersion,
		Commit:      GitCommit,
		CommitShort: GitCommitShort,
		SourceCode:  sourceCodeURL}

	log.Printf("[info] service version: %s from commit: %s (%s)", version.Version, version.CommitShort, version.Commit)
	// create database
	db := createBasketsDatabase(config.DbType, config.DbFile, config.DbConnection)
	if db == nil {
		log.Print("[error] failed to create basket database")
		return nil
	}
	createDefaultBaskets(db, config.Baskets)

	basketsDb = db

	// HTTP clients
	httpClient = new(http.Client)
	insecureTransport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	httpInsecureClient = &http.Client{Transport: insecureTransport}

	// configure service HTTP router
	router := httprouter.New()

	//// Old API mapping ////
	// basket names
	router.GET("/"+serviceOldAPIPath, GetBaskets)
	// basket management
	router.GET("/"+serviceOldAPIPath+"/:basket", GetBasket)
	router.POST("/"+serviceOldAPIPath+"/:basket", CreateBasket)
	router.PUT("/"+serviceOldAPIPath+"/:basket", UpdateBasket)
	router.DELETE("/"+serviceOldAPIPath+"/:basket", DeleteBasket)
	router.GET("/"+serviceOldAPIPath+"/:basket/responses/:method", GetBasketResponse)
	router.PUT("/"+serviceOldAPIPath+"/:basket/responses/:method", UpdateBasketResponse)
	// requests management
	router.GET("/"+serviceOldAPIPath+"/:basket/requests", GetBasketRequests)
	router.DELETE("/"+serviceOldAPIPath+"/:basket/requests", ClearBasket)

	//// New API mapping ////
	// service details
	router.GET("/"+serviceAPIPath+"/stats", GetStats)
	router.GET("/"+serviceAPIPath+"/version", GetVersion)
	// basket names
	router.GET("/"+serviceAPIPath+"/baskets", GetBaskets)
	// basket management
	router.GET("/"+serviceAPIPath+"/baskets/:basket", GetBasket)
	router.POST("/"+serviceAPIPath+"/baskets/:basket", CreateBasket)
	router.PUT("/"+serviceAPIPath+"/baskets/:basket", UpdateBasket)
	router.DELETE("/"+serviceAPIPath+"/baskets/:basket", DeleteBasket)
	router.GET("/"+serviceAPIPath+"/baskets/:basket/responses/:method", GetBasketResponse)
	router.PUT("/"+serviceAPIPath+"/baskets/:basket/responses/:method", UpdateBasketResponse)
	// requests management
	router.GET("/"+serviceAPIPath+"/baskets/:basket/requests", GetBasketRequests)
	router.DELETE("/"+serviceAPIPath+"/baskets/:basket/requests", ClearBasket)

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

func createDefaultBaskets(db BasketsDatabase, baskets []string) {
	for _, basket := range baskets {
		createDefaultBasket(db, basket)
	}
}

func createDefaultBasket(db BasketsDatabase, basket string) {
	if !validBasketName.MatchString(basket) {
		log.Printf("[error] invalid basket name to auto-create; '%s' does not match pattern: %s", basket, validBasketName.String())
	} else {
		auth, err := db.Create(basket, BasketConfig{ForwardURL: "", Capacity: serverConfig.InitCapacity})
		if err != nil {
			log.Printf("[error] %s", err)
		} else {
			log.Printf("[info] basket '%s' is auto-created with access token: %s", basket, auth.Token)
		}
	}
}
