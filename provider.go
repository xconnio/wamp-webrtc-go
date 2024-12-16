package wamp_webrtc_go

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	log "github.com/sirupsen/logrus"

	"github.com/xconnio/wampproto-go"
	"github.com/xconnio/wampproto-go/util"
	"github.com/xconnio/xconn-go"
)

type WebRTCProvider struct {
	answerers     map[string]*Answerer
	onNewAnswerer func(sessionID string, answerer *Answerer)

	sync.Mutex
}

func NewWebRTCHandler() *WebRTCProvider {
	return &WebRTCProvider{
		answerers: make(map[string]*Answerer),
	}
}

func (r *WebRTCProvider) OnAnswerer(callback func(sessionID string, answerer *Answerer)) {
	r.Lock()
	defer r.Unlock()

	r.onNewAnswerer = callback
}

func (r *WebRTCProvider) ensureAnswerer(sessionID string) *Answerer {
	r.Lock()
	defer r.Unlock()

	answerer, exists := r.answerers[sessionID]
	if !exists {
		answerer = NewAnswerer()
		r.answerers[sessionID] = answerer
		if r.onNewAnswerer != nil {
			r.onNewAnswerer(sessionID, answerer)
		}
	}

	return answerer
}

func (r *WebRTCProvider) addIceCandidate(requestID string, candidate webrtc.ICECandidateInit) error {
	answerer := r.ensureAnswerer(requestID)
	return answerer.AddICECandidate(candidate)
}

func (r *WebRTCProvider) handleOffer(requestID string, offer Offer, answerConfig *AnswerConfig) (*Answer, error) {
	answerer := r.ensureAnswerer(requestID)
	return answerer.Answer(answerConfig, offer, 100*time.Millisecond)
}

func (r *WebRTCProvider) Setup(config *ProviderConfig) {
	_, err := config.Session.Register(config.ProcedureHandleOffer, r.offerFunc, nil)
	if err != nil {
		log.Errorf("failed to register webrtc offer: %v", err)
		return
	}

	_, err = config.Session.Subscribe(config.TopicHandleRemoteCandidates, r.onRemoteCandidate, nil)
	if err != nil {
		log.Errorf("failed to subscribe to webrtc candidates events: %v", err)
		return
	}

	r.OnAnswerer(func(sessionID string, answerer *Answerer) {
		answerer.OnIceCandidate(func(candidate *webrtc.ICECandidate) {
			answerData, err := json.Marshal(candidate.ToJSON())
			if err != nil {
				log.Errorf("failed to marshal answer: %v", err)
				return
			}

			args := []any{sessionID, string(answerData)}
			if err = config.Session.Publish(config.TopicPublishLocalCandidate, args, nil, nil); err != nil {
				log.Errorf("failed to publish answer: %v", err)
			}
		})

		go func() {
			select {
			case channel := <-answerer.WaitReady():
				if err = r.handleWAMPClient(channel, config); err != nil {
					log.Errorf("failed to handle answer: %v", err)
					_ = answerer.connection.Close()
				}
			case <-time.After(20 * time.Second):
				log.Errorln("webrtc connection didn't establish after 20 seconds")
			}
		}()
	})
}

func (r *WebRTCProvider) handleWAMPClient(channel *webrtc.DataChannel, config *ProviderConfig) error {
	rtcPeer := NewWebRTCPeer(channel)

	hello, err := xconn.ReadHello(rtcPeer, config.Serializer)
	if err != nil {
		return err
	}

	base, err := xconn.Accept(rtcPeer, hello, config.Serializer, config.Authenticator)
	if err != nil {
		return err
	}

	if !config.Routed {
		return nil
	}

	xconnRouter := xconn.NewRouter()
	xconnRouter.AddRealm("realm1")
	if err = xconnRouter.AttachClient(base); err != nil {
		return fmt.Errorf("failed to attach client %w", err)
	}

	parser := NewWebRTCMessageAssembler()
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fullMsg := parser.Feed(msg.Data)

		if fullMsg != nil {
			if err = base.Write(fullMsg); err != nil {
				log.Errorf("failed to send wamp message: %v", err)
				return
			}
		}
	})

	channel.OnClose(func() {
		_ = base.Close()
	})

	for {
		msg, err := base.ReadMessage()
		if err != nil {
			_ = xconnRouter.DetachClient(base)
			break
		}

		if err = xconnRouter.ReceiveMessage(base, msg); err != nil {
			log.Println(err)
			return nil
		}

		data, err := config.Serializer.Serialize(msg)
		if err != nil {
			log.Printf("failed to serialize message: %v", err)
			return nil
		}

		for chunk := range parser.ChunkMessage(data) {
			if err = channel.Send(chunk); err != nil {
				log.Errorf("failed to write message: %v", err)
				return nil
			}
		}
	}

	return err
}

func (r *WebRTCProvider) offerFunc(_ context.Context, invocation *xconn.Invocation) *xconn.Result {
	if len(invocation.Arguments) < 2 {
		return &xconn.Result{Err: wampproto.ErrInvalidArgument}
	}

	requestID, ok := util.AsString(invocation.Arguments[0])
	if !ok {
		return &xconn.Result{Err: wampproto.ErrInvalidArgument, Arguments: []any{"request ID must be a string"}}
	}

	offerJSON, ok := util.AsString(invocation.Arguments[1])
	if !ok {
		return &xconn.Result{Err: wampproto.ErrInvalidArgument, Arguments: []any{"offer must be a string"}}
	}

	var offer Offer
	if err := json.Unmarshal([]byte(offerJSON), &offer); err != nil {
		return &xconn.Result{Err: wampproto.ErrInvalidArgument, Arguments: []any{err.Error()}}
	}

	cfg := &AnswerConfig{ICEServers: []webrtc.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	}}
	answer, err := r.handleOffer(requestID, offer, cfg)
	if err != nil {
		return &xconn.Result{Err: wampproto.ErrInvalidArgument, Arguments: []any{err.Error()}}
	}

	answerData, err := json.Marshal(answer)
	if err != nil {
		return &xconn.Result{Err: wampproto.ErrInvalidArgument, Arguments: []any{err.Error()}}
	}

	return &xconn.Result{Arguments: []any{string(answerData)}}
}

func (r *WebRTCProvider) onRemoteCandidate(event *xconn.Event) {
	if len(event.Arguments) < 2 {
		return
	}

	requestID, ok := util.AsString(event.Arguments[0])
	if !ok {
		log.Errorln("request ID must be a string")
		return
	}

	candidateJSON, ok := event.Arguments[1].(string)
	if !ok {
		log.Errorln("offer must be a string")
		return
	}

	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal([]byte(candidateJSON), &candidate); err != nil {
		return
	}

	if err := r.addIceCandidate(requestID, candidate); err != nil {
		log.Errorf("failed to add ice candidate: %v", err)
		return
	}
}
