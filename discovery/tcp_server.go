package discovery

import (
	"encoding/json"
	"log"
	"net"
	"sync"
)

// TCPServer represents a TCP server for peer connection
type TCPServer struct {
	address  string
	listener net.Listener
	clients  map[string]net.Conn
	mu       sync.Mutex
	router   *MessageRouter
}

// NewTCPServer creates a new TCP server instance
func NewTCPServer(address string, router *MessageRouter) *TCPServer {
	return &TCPServer{
		address: address,
		clients: map[string]net.Conn{},
		router:  router,
	}
}

// Start begins listening for incoming TCP connections
func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.listener = listener
	log.Println("[TCP] Server started on ", s.address)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
					log.Println("[TCP] Server shutting down")
					return
				}
				log.Println("[TCP] Error accepting connection: ", err)
				continue
			}
			go s.handleClient(conn)
		}
	}()
	return nil
}

// handleClient reads messages from a client and calls the user defined handler
func (s *TCPServer) handleClient(conn net.Conn) {
	defer func() {
		s.removeClient(conn)
		conn.Close()
	}()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	peerId := conn.RemoteAddr().String() // Use connection address as ID

	s.mu.Lock()
	s.clients[peerId] = conn
	s.mu.Unlock()

	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			log.Println("[TCP] Error decoding message from peer ", peerId, " : ", err)
			return
		}

		response := s.router.HandleMessage(peerId, msg)

		if err := encoder.Encode(response); err != nil {
			log.Println("[TCP] Error encoding response to peer ", peerId, " : ", err)
			return
		}
	}
}

// removeClient safely removes a client connection
func (s *TCPServer) removeClient(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, c := range s.clients {
		if c == conn {
			delete(s.clients, id)
			log.Println("[TCP] Client ", id, " disconnected")
			break
		}
	}
}

// Stop gracefully shuts down the server
func (s *TCPServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}
	for _, conn := range s.clients {
		conn.Close()
	}
	s.clients = make(map[string]net.Conn)
	log.Println("[TCP] Server stopped")
}
