package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/xconnio/wamp-webrtc-go"
	"github.com/xconnio/xconn-go"
)

const (
	procedureWebRTCOffer     = "io.xconn.webrtc.offer"
	topicAnswererOnCandidate = "io.xconn.webrtc.answerer.on_candidate"
)

func main() {
	config := &wamp_webrtc_go.ClientConfig{
		URL:                      "ws://localhost:8080/ws",
		Realm:                    "realm1",
		ProcedureWebRTCOffer:     procedureWebRTCOffer,
		TopicAnswererOnCandidate: topicAnswererOnCandidate,
		Serializer:               xconn.CBORSerializerSpec,
	}
	session, err := wamp_webrtc_go.ConnectWebRTC(config)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(session)
}
