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
	pathPrefix := getPathPrefix(config)
	router := httprouter.New()

	//// Old API mapping ////
	// basket names
	router.GET(pathPrefix+"/"+serviceOldAPIPath, GetBaskets)
	// basket management
	router.GET(pathPrefix+"/"+serviceOldAPIPath+"/:basket", GetBasket)
	router.POST(pathPrefix+"/"+serviceOldAPIPath+"/:basket", CreateBasket)
	router.PUT(pathPrefix+"/"+serviceOldAPIPath+"/:basket", UpdateBasket)
	router.DELETE(pathPrefix+"/"+serviceOldAPIPath+"/:basket", DeleteBasket)
	router.GET(pathPrefix+"/"+serviceOldAPIPath+"/:basket/responses/:method", GetBasketResponse)
	router.PUT(pathPrefix+"/"+serviceOldAPIPath+"/:basket/responses/:method", UpdateBasketResponse)
	// requests management
	router.GET(pathPrefix+"/"+serviceOldAPIPath+"/:basket/requests", GetBasketRequests)
	router.DELETE(pathPrefix+"/"+serviceOldAPIPath+"/:basket/requests", ClearBasket)

	//// New API mapping ////
	// service details
	router.GET(pathPrefix+"/"+serviceAPIPath+"/stats", GetStats)
	router.GET(pathPrefix+"/"+serviceAPIPath+"/version", GetVersion)
	// basket names
	router.GET(pathPrefix+"/"+serviceAPIPath+"/baskets", GetBaskets)
	// basket management
	router.GET(pathPrefix+"/"+serviceAPIPath+"/baskets/:basket", GetBasket)
	router.POST(pathPrefix+"/"+serviceAPIPath+"/baskets/:basket", CreateBasket)
	router.PUT(pathPrefix+"/"+serviceAPIPath+"/baskets/:basket", UpdateBasket)
	router.DELETE(pathPrefix+"/"+serviceAPIPath+"/baskets/:basket", DeleteBasket)
	router.GET(pathPrefix+"/"+serviceAPIPath+"/baskets/:basket/responses/:method", GetBasketResponse)
	router.PUT(pathPrefix+"/"+serviceAPIPath+"/baskets/:basket/responses/:method", UpdateBasketResponse)
	// requests management
	router.GET(pathPrefix+"/"+serviceAPIPath+"/baskets/:basket/requests", GetBasketRequests)
	router.DELETE(pathPrefix+"/"+serviceAPIPath+"/baskets/:basket/requests", ClearBasket)

	// web pages
	router.GET(pathPrefix+"/", ForwardToWeb)
	router.GET(pathPrefix+"/"+serviceUIPath, WebIndexPage)
	router.GET(pathPrefix+"/"+serviceUIPath+"/:basket", WebBasketPage)

	// static assets serving
	router.ServeFiles(pathPrefix+"/static/*filepath", http.Dir("./static"))

	// basket requests
	router.NotFound = http.HandlerFunc(AcceptBasketRequests)

	log.Printf("[info] HTTP server is listening on http://%s:%d", serverConfig.ServerAddr, serverConfig.ServerPort)
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

func getPathPrefix(config *ServerConfig) string {
	pathPrefix := config.PathPrefix
	if len(pathPrefix) > 0 {
		log.Printf("[info] service path prefix: %s", pathPrefix)
	}

	return pathPrefix
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
