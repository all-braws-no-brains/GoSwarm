package discovery

import (
	"encoding/json"
	"net"
	"testing"
)

func TestTCPServer_BasicMessageExchange(t *testing.T) {
	peerStore := NewPeerStore(0)
	router := NewMessageHandler(peerStore)

	// Use port 0 to get an available port dynamically
	server := NewTCPServer("0.0.0.0:8000", router)

	ready := make(chan struct{})
	errChan := make(chan error, 1)

	go func() {
		if err := server.Start(); err != nil {
			errChan <- err
		}
		close(ready)
	}()

	defer server.Stop()

	select {
	case <-ready:
	case err := <-errChan:
		t.Fatalf("Failed to start TCP server: %v", err)
	}
	server.mu.Lock()
	serverAddr := server.listener.Addr().String()
	server.mu.Unlock()

	// Attempt to connect
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to connect to TCP server: %v", err)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	// Send a message
	msg := NewMessage("PING", "client1", "server", nil, false)
	if err := encoder.Encode(msg); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	var response Message
	if err := decoder.Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Type != "PONG" {
		t.Errorf("Expected PONG response, got %s", response.Type)
	}
}
