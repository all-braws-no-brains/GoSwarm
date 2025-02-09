package discovery

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

// routingTable maintains peer routes internally
type routingTable struct {
	mu    sync.RWMutex
	peers map[string]string
}

func newRoutingTable() *routingTable {
	return &routingTable{
		peers: make(map[string]string),
	}
}

func (rt *routingTable) addPeer(id, address string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.peers[id] = address
}

func (rt *routingTable) getPeer(id string) (string, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	address, exists := rt.peers[id]
	return address, exists
}

func (rt *routingTable) removePeer(id string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	delete(rt.peers, id)
}

func (rt *routingTable) getAllPeers() map[string]string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	peersCopy := make(map[string]string, len(rt.peers))
	for id, addrs := range rt.peers {
		peersCopy[id] = addrs
	}
	return peersCopy
}

type PeerRouter struct {
	listener     net.Listener
	mu           sync.Mutex
	routingTable *routingTable
}

// NewPeerRouter initializes the router.
func NewPeerRouter() *PeerRouter {
	return &PeerRouter{
		routingTable: newRoutingTable(),
	}
}

// RegisterPeer adds a peer to the routing system (exposed for users).
func (pr *PeerRouter) RegisterPeer(id, address string) {
	pr.routingTable.addPeer(id, address)
}

// UnregisterPeer removes a peer from the routing system (exposed for users).
func (pr *PeerRouter) UnregisterPeer(id string) {
	pr.routingTable.removePeer(id)
}

// ResolvePeer retrieves a peer's address (exposed for users if needed).
func (pr *PeerRouter) ResolvePeer(id string) (string, error) {
	address, exists := pr.routingTable.getPeer(id)
	if !exists {
		return "", errors.New("peer not found")
	}
	return address, nil
}

// BrodcastMessage sends a message to all available peers
func (pr *PeerRouter) BrodcastMessage(data []byte) {
	peers := pr.routingTable.getAllPeers()

	var wg sync.WaitGroup
	for id, addr := range peers {
		wg.Add(1)
		go func(id, addr string) {
			defer wg.Done()
			if err := pr.sendMessage(addr, data); err != nil {
				log.Printf("[BRODCAST] Failed to send message to %s (%s): %v", id, addr, err)
			}
		}(id, addr)
	}
	wg.Wait()
}

// SendMessage sends a message to a specific peer
func (pr *PeerRouter) SendMessage(peerId string, data []byte) error {
	address, exists := pr.routingTable.getPeer(peerId)
	if !exists {
		return errors.New("peer not found in routing table")
	}
	return pr.sendMessage(address, data)
}

func (pr *PeerRouter) sendMessage(address string, data []byte) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(data)
	return err
}

// StartListener starts a TCP server to listen for incoming messages
func (pr *PeerRouter) StartListener(port int) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.listener != nil {
		return errors.New("listener aready running")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	pr.listener = listener
	log.Printf("[PeerRouter] Listening on port: %d", port)

	go pr.acceptConnections()
	return nil
}

func (pr *PeerRouter) acceptConnections() {
	for {
		conn, err := pr.listener.Accept()
		if err != nil {
			log.Println("[PeerRouter] Error accepting connection: ", err)
			continue
		}

		go pr.handleConnection(conn)
	}
}

func (pr *PeerRouter) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadBytes('\n') // read until new line
		if err != nil {
			log.Println("[PeerRouter] Error reading message: ", err)
			return
		}

		log.Printf("[PeerRouter] Received message: %s", message)
	}
}

// StopListener shuts down the TCP Listener
func (pr *PeerRouter) StopListener() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.listener == nil {
		return errors.New("listener not running")
	}

	err := pr.listener.Close()
	pr.listener = nil
	return err
}

// MessageHandler defines the function signature for processing messages
type MessageHandler func(peerId, message string) string

// MessageRouter manages built-in and custom message handlers
type MessageRouter struct {
	mu        sync.Mutex
	handlers  map[string]MessageHandler
	peerStore *PeerStore
}

// NewMessageHandler initializes a message router with built-in handlers
func NewMessageHandler(peerStore *PeerStore) *MessageRouter {
	router := &MessageRouter{
		handlers:  make(map[string]MessageHandler),
		peerStore: peerStore,
	}
	router.RegisterHandler("PING", func(peerId, msg string) string {
		return "PONG"
	})

	router.RegisterHandler("ECHO", func(peerId, msg string) string {
		return msg
	})

	router.RegisterHandler("PEER_LIST", func(peerId, msg string) string {
		peers := router.peerStore.GetPeers()
		if len(peers) == 0 {
			return "No Active Peers..."
		}

		var peerList []string
		for _, p := range peers {
			peerList = append(peerList, fmt.Sprintf("%s:%d", p.Address, p.Port))
		}
		return strings.Join(peerList, ", ")
	})

	return router
}

// RegisterHandler allows users to define custom message handlers
func (mr *MessageRouter) RegisterHandler(command string, handler MessageHandler) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.handlers[strings.ToUpper(command)] = handler
}

// HandleMessage processs an incoming message an returns a response
func (mr *MessageRouter) HandleMessage(peerId, message string) string {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	parts := strings.Split(message, "|")
	command := strings.ToUpper(parts[0])

	if handler, exists := mr.handlers[command]; exists {
		data := ""
		if len(parts) > 1 {
			data = parts[1]
		}
		return handler(peerId, data)
	}
	return fmt.Sprintf("ERROR: Unknown command '%s'", command)
}
