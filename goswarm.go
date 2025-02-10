package discovery

import (
	"log"
	"time"

	"github.com/all-braws-no-brains/GoSwarm/discovery"
)

// GoSwarm is the main entry point for users of the library.
type GoSwarm struct {
	peerStore   *discovery.PeerStore
	peerRouter  *discovery.PeerRouter
	msgRouter   *discovery.MessageRouter
	peerTracker *discovery.PeerTracker
	tcpServer   *discovery.TCPServer
	mdns        *discovery.MDNSService
}

// NewGoSwarm initializes the GoSwarm instance and its components.
func NewGoSwarm(address, dbFile string, cleanupInterval, cleanupThreshold time.Duration) (*GoSwarm, error) {
	peerStore := discovery.NewPeerStore(0)
	msgRouter := discovery.NewMessageHandler(peerStore)
	peerRouter := discovery.NewPeerRouter(msgRouter)
	peerTracker, err := discovery.NewPeerTracker(peerStore, dbFile, cleanupInterval)
	if err != nil {
		return nil, err
	}
	tcpServer := discovery.NewTCPServer(address, msgRouter)
	mdns := discovery.NewMDNSService("_goswarm._tcp")

	swarm := &GoSwarm{
		peerStore:   peerStore,
		peerRouter:  peerRouter,
		msgRouter:   msgRouter,
		peerTracker: peerTracker,
		tcpServer:   tcpServer,
		mdns:        mdns,
	}

	return swarm, nil
}

// Start launches all core services of GoSwarm.
func (s *GoSwarm) Start() error {
	log.Println("[GoSwarm] Starting services...")

	go s.tcpServer.Start()
	go s.mdns.Start(8000)                                        // Example port
	go s.peerTracker.StartCleanup(30*time.Minute, 5*time.Minute) // Example cleanup interval

	return nil
}

// Stop gracefully shuts down all GoSwarm components.
func (s *GoSwarm) Stop() {
	log.Println("[GoSwarm] Stopping services...")
	s.tcpServer.Stop()
	s.mdns.Stop()
	s.peerTracker.StopCleanup()
}

// SendMessage sends a message to a specific peer.
func (s *GoSwarm) SendMessage(peerID string, data []byte, isBinary bool) error {
	msg := discovery.NewMessage("DATA", "self", peerID, data, isBinary)
	return s.peerRouter.SendMessage(peerID, msg)
}

// Broadcast sends a message to all connected peers.
func (s *GoSwarm) Broadcast(data []byte, isBinary bool) {
	msg := discovery.NewMessage("BROADCAST", "self", "", data, isBinary)
	s.peerRouter.BroadcastMessage(msg)
}

// RegisterPeer manually registers a peer in the routing table.
func (s *GoSwarm) RegisterPeer(id, address string) {
	s.peerRouter.RegisterPeer(id, address)
}

// UnregisterPeer removes a peer from the routing table.
func (s *GoSwarm) UnregisterPeer(id string) {
	s.peerRouter.UnregisterPeer(id)
}

// ResolvePeer retrieves the address of a peer.
func (s *GoSwarm) ResolvePeer(id string) (string, error) {
	return s.peerRouter.ResolvePeer(id)
}

// GetAllPeers returns a list of all known peers.
func (s *GoSwarm) GetAllPeers() map[string]string {
	return s.peerRouter.GetAllPeers()
}
