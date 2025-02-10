package discovery

import "testing"

func TestMessageRouter_BUILTIN(t *testing.T) {
	peerStore := NewPeerStore(0)
	router := NewMessageHandler(peerStore)

	resp := router.HandleMessage("peer_1", NewMessage("PING", "peer_1", "server", nil, false))
	if resp.Type != "PONG" {
		t.Errorf("Expected PONG, got %s", resp.Type)
	}

	msg := NewMessage("ECHO", "peer1", "server", []byte("hello"), false)
	resp = router.HandleMessage("peer1", msg)
	if string(resp.Payload) != "hello" {
		t.Errorf("Expected 'hello', got %s", string(resp.Payload))
	}

	resp = router.HandleMessage("peer1", NewMessage("PEER_LIST", "peer1", "server", nil, false))
	if string(resp.Payload) != "No Active Peers..." {
		t.Errorf("Expected 'No Active Peers...', got %s", string(resp.Payload))
	}
}

func TestMessageRouter_CUSTOM_HANDLER(t *testing.T) {
	peerStore := NewPeerStore(0)
	router := NewMessageHandler(peerStore)

	router.RegisterHandler("CUSTOM", func(peerId string, msg Message) Message {
		return NewMessage("CUSTOM_RESPONSE", "server", peerId, []byte("Custom Success"), false)
	})

	resp := router.HandleMessage("peer1", NewMessage("CUSTOM", "peer1", "server", nil, false))
	if resp.Type != "CUSTOM_RESPONSE" || string(resp.Payload) != "Custom Success" {
		t.Errorf("Custom handler failed, got: %s, payload: %s", resp.Type, string(resp.Payload))
	}
}

func TestMessageRouter_UNKNOWN_COMMAND(t *testing.T) {
	peerStore := NewPeerStore(0)
	router := NewMessageHandler(peerStore)

	resp := router.HandleMessage("peer1", NewMessage("INVALID_CMD", "peer1", "server", nil, false))
	if resp.Type != "ERROR" {
		t.Errorf("Expected ERROR response, got %s", resp.Type)
	}
}
