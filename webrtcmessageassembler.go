package wamp_webrtc_go

import (
	"bytes"
	"sync"
)

type WebRTCMessageAssembler struct {
	buffer *bytes.Buffer

	sync.Mutex
}

func NewWebRTCMessageAssembler() *WebRTCMessageAssembler {
	return &WebRTCMessageAssembler{
		buffer: bytes.NewBuffer(nil),
	}
}

func (m *WebRTCMessageAssembler) ChunkMessage(message []byte) chan []byte {
	m.Lock()
	defer m.Unlock()

	chunkSize := 16*1024 - 1
	totalChunks := (len(message) + chunkSize - 1) / chunkSize

	chunks := make(chan []byte)

	go func() {
		for i := 0; i < totalChunks; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if i == totalChunks-1 {
				end = len(message)
			}
			chunk := message[start:end]

			var isFinal byte = 0
			if i == totalChunks-1 {
				isFinal = 1
			}

			chunks <- append([]byte{isFinal}, chunk...)
		}
		close(chunks)
	}()

	return chunks
}

func (m *WebRTCMessageAssembler) Feed(data []byte) []byte {
	m.Lock()
	defer m.Unlock()

	m.buffer.Write(data[1:])
	isFinal := data[0]
	if isFinal == 1 {
		out := m.buffer.Bytes()
		m.buffer.Reset()
		return out
	}

	return nil
}
