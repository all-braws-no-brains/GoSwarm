package discovery

import (
	"encoding/json"
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
	listener      net.Listener
	mu            sync.Mutex
	routingTable  *routingTable
	messageRouter *MessageRouter
}

// NewPeerRouter initializes the router.
func NewPeerRouter(messageRouter *MessageRouter) *PeerRouter {
	return &PeerRouter{
		routingTable:  newRoutingTable(),
		messageRouter: messageRouter,
	}
}

// GetAllPeers returns a copy of all registered peers.
func (pr *PeerRouter) GetAllPeers() map[string]string {
	return pr.routingTable.getAllPeers()
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

// BroadcastMessage sends a message to all available peers
func (pr *PeerRouter) BroadcastMessage(msg Message) {
	peers := pr.routingTable.getAllPeers()

	var wg sync.WaitGroup
	for id, addr := range peers {
		wg.Add(1)
		go func(id, addr string) {
			defer wg.Done()
			if err := pr.sendMessage(addr, msg); err != nil {
				log.Printf("[BRODCAST] Failed to send message to %s (%s): %v", id, addr, err)
			}
		}(id, addr)
	}
	wg.Wait()
}

// SendMessage sends a message to a specific peer
func (pr *PeerRouter) SendMessage(peerId string, msg Message) error {
	address, exists := pr.routingTable.getPeer(peerId)
	if !exists {
		return errors.New("peer not found in routing table")
	}
	return pr.sendMessage(address, msg)
}

func (pr *PeerRouter) sendMessage(address string, msg Message) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	return encoder.Encode(msg)
}

// StartListener starts a TCP server to listen for incoming messages
func (pr *PeerRouter) StartListener(port int) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.listener != nil {
		return errors.New("listener already running")
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

	decoder := json.NewDecoder(conn)
	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			log.Println("[PeerRouter] Failed to decode message: ", err)
			return
		}
		log.Printf("[PeerRouter] Received message from %s: %+v", msg.SenderId, msg)

		if response := pr.routeMessage(msg); response != nil {
			encoder := json.NewEncoder(conn)
			encoder.Encode(response)
		}
	}
}

// routeMessage processes incoming messages and determines weather it has to be routed or not
func (pr *PeerRouter) routeMessage(msg Message) *Message {
	// if the message has a specific receiver, proceed to route it to said receiver
	if msg.ReceiverId != "" {
		if err := pr.SendMessage(msg.ReceiverId, msg); err != nil {
			log.Printf("[PeerRouter] Failed to forward message to %s: %v\n", msg.ReceiverId, err)
			return &Message{
				Type:       "ERROR",
				SenderId:   "server",
				ReceiverId: msg.SenderId,
				Payload:    []byte(fmt.Sprintf("Failed to deliver message: %v", err)),
				IsBinary:   false,
			}
		}
		return nil
	}

	if pr.messageRouter != nil {
		response := pr.messageRouter.HandleMessage(msg.SenderId, msg)
		return &response
	}

	// Error if no router available
	return &Message{
		Type:       "ERROR",
		SenderId:   "server",
		ReceiverId: msg.SenderId,
		Payload:    []byte("No routing handler available."),
		IsBinary:   false,
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
type MessageHandler func(peerId string, msg Message) Message

// MessageRouter manages built-in and custom message handlers
type MessageRouter struct {
	mu        sync.RWMutex
	handlers  map[string]MessageHandler
	peerStore *PeerStore
}

// NewMessageHandler initializes a message router with built-in handlers
func NewMessageHandler(peerStore *PeerStore) *MessageRouter {
	router := &MessageRouter{
		handlers:  make(map[string]MessageHandler),
		peerStore: peerStore,
	}

	router.RegisterHandler("PING", func(peerId string, msg Message) Message {
		return NewMessage("PONG", "server", peerId, nil, msg.IsBinary)
	})

	router.RegisterHandler("ECHO", func(peerId string, msg Message) Message {
		return NewMessage("ECHO", "server", peerId, msg.Payload, msg.IsBinary)
	})

	router.RegisterHandler("PEER_LIST", func(peerId string, msg Message) Message {
		peers := router.peerStore.GetPeers()
		if len(peers) == 0 {
			return NewMessage("PEER_LIST", "server", peerId, []byte("No Active Peers..."), msg.IsBinary)
		}

		var peerList []string
		for _, p := range peers {
			peerList = append(peerList, fmt.Sprintf("%s:%d", p.Address, p.Port))
		}
		return NewMessage("PEER_LIST", "server", peerId, []byte(strings.Join(peerList, ", ")), msg.IsBinary)
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
func (mr *MessageRouter) HandleMessage(peerId string, msg Message) Message {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	// Find the handler based on message type
	if handler, exists := mr.handlers[strings.ToUpper(msg.Type)]; exists {
		return handler(peerId, msg)
	}

	return NewMessage("ERROR", "server", peerId, []byte(fmt.Sprintf("Unknown command '%s'", msg.Type)), false)
}
