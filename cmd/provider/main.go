package main

import (
	"context"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"

	"github.com/xconnio/wamp-webrtc-go"
	"github.com/xconnio/wampproto-go/serializers"
	"github.com/xconnio/xconn-go"
)

const (
	procedureWebRTCOffer     = "io.xconn.webrtc.offer"
	topicOffererOnCandidate  = "io.xconn.webrtc.offerer.on_candidate"
	topicAnswererOnCandidate = "io.xconn.webrtc.answerer.on_candidate"
)

func main() {
	session, err := xconn.Connect(context.Background(), "ws://localhost:8080/ws", "realm1")
	if err != nil {
		log.Fatal("Failed to connect to server:", err)
	}

	webRtcManager := wamp_webrtc_go.NewWebRTCHandler()
	cfg := &wamp_webrtc_go.ProviderConfig{
		Session:                     session,
		ProcedureHandleOffer:        procedureWebRTCOffer,
		TopicHandleRemoteCandidates: topicOffererOnCandidate,
		TopicPublishLocalCandidate:  topicAnswererOnCandidate,
		Serializer:                  &serializers.CBORSerializer{},
	}
	webRtcManager.Setup(cfg)

	// Close server if SIGINT (CTRL-c) received.
	closeChan := make(chan os.Signal, 1)
	signal.Notify(closeChan, os.Interrupt)

	select {
	case <-closeChan:
	case <-session.LeaveChan():
	}
}
