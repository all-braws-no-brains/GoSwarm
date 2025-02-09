package communication

import (
	"log"
	"net"
	"sync"
)

// TCPServer represents a TCP server for peer connection
type TCPServer struct {
	address   string
	listener  net.Listener
	clients   map[string]net.Conn
	mu        sync.Mutex
	onMessage func(conn net.Conn, message []byte)
}

// NewTCPServer creates a new TCP server instance
func NewTCPServer(address string, onMessage func(conn net.Conn, message []byte)) *TCPServer {
	return &TCPServer{
		address:   address,
		clients:   map[string]net.Conn{},
		onMessage: onMessage,
	}
}

// Start begins listening for incoming TCP connections
func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.listener = listener

	log.Println("TCP server started on ", s.address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection: ", err)
			continue
		}

		s.mu.Lock()
		s.clients[conn.RemoteAddr().String()] = conn
		s.mu.Unlock()

		go s.handleClient(conn)
	}
}

// handleClient reads messages from a client and calls the user defined handler
func (s *TCPServer) handleClient(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn.RemoteAddr().String())
		s.mu.Unlock()
		conn.Close()
	}()

	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println("Client disconnected: ", conn.RemoteAddr().String())
			return
		}

		if s.onMessage != nil {
			s.onMessage(conn, buffer[:n])
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
	log.Println("TCP Server stopped")
}
