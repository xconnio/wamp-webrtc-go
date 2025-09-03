package main

import (
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"

	"github.com/xconnio/xconn-go"
)

func main() {
	r := xconn.NewRouter()
	if err := r.AddRealm("realm1"); err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	server := xconn.NewServer(r, nil, nil)
	closer, err := server.ListenAndServeWebSocket(xconn.NetworkTCP, "0.0.0.0:8080")
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
	defer closer.Close()

	// Close server if SIGINT (CTRL-c) received.
	closeChan := make(chan os.Signal, 1)
	signal.Notify(closeChan, os.Interrupt)
	<-closeChan
}
