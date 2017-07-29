package main

import "log"

var serverConfig *ServerConfig

func main() {
	// read config
	serverConfig = CreateConfig()
	// create & start server
	if server := CreateServer(serverConfig); server != nil {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}
}
