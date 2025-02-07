package discovery

import (
	"sync"
	"time"
)

// peer (private) represents a discovered node in the P2P network
type peer struct {
	id       string
	address  string
	lastSeen time.Time
	port     int
}

// Peer (public) is the exported struct for returning peer data safely
type Peer struct {
	ID       string
	Address  string
	LastSeen time.Time
	Port     int
}

// PeerStore manages a list pf discoverd peers securely
type PeerStore struct {
	mu    sync.RWMutex
	peers map[string]*peer
}

// NewPeerStore initializes and returns a secure PeerStore
func NewPeerStore() *PeerStore {
	return &PeerStore{
		peers: make(map[string]*peer),
	}
}

// AddPeer safely adds a new peer into the store
func (ps *PeerStore) AddPeer(id, address string, port int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if p, exists := ps.peers[id]; exists {
		p.lastSeen = time.Now()
	} else {
		ps.peers[id] = &peer{
			id:       id,
			address:  address,
			lastSeen: time.Now(),
			port:     port,
		}
	}
}

func (ps *PeerStore) GetPeers() []Peer {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var peers []Peer
	for _, p := range ps.peers {
		peers = append(peers, Peer{
			ID:       p.id,
			Address:  p.address,
			LastSeen: p.lastSeen,
			Port:     p.port,
		})
	}
	return peers
}

// Cleanup removes peers that have been inactive beyond a certain threshold.
func (ps *PeerStore) Cleanup(expiry time.Duration) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	for id, p := range ps.peers {
		if now.Sub(p.lastSeen) > expiry {
			delete(ps.peers, id)
		}
	}
}
