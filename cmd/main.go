package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/xconnio/xconn-go"
)

func main() {
	r := xconn.NewRouter()
	r.AddRealm("realm1")

	server := xconn.NewServer(r, nil, nil)
	closer, err := server.Start("localhost", 8080)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer closer.Close()

	closeChan := make(chan os.Signal, 1)
	signal.Notify(closeChan, os.Interrupt)
	<-closeChan
}
