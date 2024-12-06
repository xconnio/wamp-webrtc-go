package wamp_webrtc_go

import (
	"github.com/pion/webrtc/v4"

	"github.com/xconnio/wampproto-go/serializers"
	"github.com/xconnio/xconn-go"
)

type Answer struct {
	Candidates  []webrtc.ICECandidateInit `json:"candidates"`
	Description webrtc.SessionDescription `json:"description"`
}

type Offer = Answer

type OfferConfig struct {
	Protocol                 string
	ICEServers               []webrtc.ICEServer
	Ordered                  bool
	ID                       uint16
	TopicAnswererOnCandidate string
}

type AnswerConfig struct {
	ICEServers []webrtc.ICEServer
}

type ProviderConfig struct {
	Session                     *xconn.Session
	ProcedureHandleOffer        string
	TopicHandleRemoteCandidates string
	TopicPublishLocalCandidate  string
	Serializer                  serializers.Serializer
}

type WebRTCSession struct {
	Connection *webrtc.PeerConnection
	Channel    *webrtc.DataChannel
}
