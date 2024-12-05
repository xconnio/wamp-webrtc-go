package wamp_webrtc_go

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"

	"github.com/xconnio/xconn-go"
)

type ClientConfig struct {
	URL                      string
	Realm                    string
	ProcedureWebRTCOffer     string
	TopicAnswererOnCandidate string
	Serializer               xconn.WSSerializerSpec
}

func ConnectWebRTC(config *ClientConfig) (*xconn.Session, error) {
	session, err := xconn.Connect(context.Background(), config.URL, config.Realm)
	if err != nil {
		return nil, err
	}

	offerer := NewOfferer()
	offerConfig := &OfferConfig{
		Protocol:                 config.Serializer.SubProtocol(),
		ICEServers:               []webrtc.ICEServer{},
		Ordered:                  true,
		TopicAnswererOnCandidate: config.TopicAnswererOnCandidate,
	}
	requestID := uuid.New().String()
	offer, err := offerer.Offer(offerConfig, session, requestID)
	if err != nil {
		return nil, err
	}

	offerJSON, err := json.Marshal(offer)
	if err != nil {
		return nil, err
	}

	result, err := session.Call(context.Background(), config.ProcedureWebRTCOffer,
		[]any{requestID, string(offerJSON)}, nil, nil)
	if err != nil {
		return nil, err
	}

	answerText := result.Arguments[0].(string)
	var answer Answer
	if err = json.Unmarshal([]byte(answerText), &answer); err != nil {
		return nil, err
	}

	if err = offerer.HandleAnswer(answer); err != nil {
		return nil, err
	}

	channel := <-offerer.WaitReady()

	peer := NewWebRTCPeer(channel)
	base, err := xconn.Join(peer, config.Realm, config.Serializer.Serializer(), nil)
	if err != nil {
		return nil, err
	}

	wampSession := xconn.NewSession(base, config.Serializer.Serializer())

	return wampSession, nil
}
