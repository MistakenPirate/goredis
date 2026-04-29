package main

import (
	"flag"
	"log"

	"github.com/mistakenpirate/goredis/config"
	"github.com/mistakenpirate/goredis/server"
)


func setupFlags(){
	flag.StringVar(&config.Host,"host","0.0.0.0","host for the redis server")
	flag.IntVar(&config.Port,"port",7379,"port for the redis server")
	flag.Parse()
}

func main(){
	setupFlags()
	log.Println("initiating the server")
	server.RunSyncTCPServer()
}