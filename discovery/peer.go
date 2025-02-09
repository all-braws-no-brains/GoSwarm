package discovery

import (
	"encoding/json"
	"sync"
	"time"
)

// peer (private) represents a discovered node in the P2P network
type peer struct {
	id       string
	address  string
	lastSeen time.Time
	port     int
	stats    *PeerStats
}

func (p *peer) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID       string    `json:"id"`
		Address  string    `json:"address"`
		LastSeen time.Time `json:"last_seen"`
		Port     int       `json:"port"`
		Stats    PeerStats `json:"stats"`
	}{
		ID:       p.id,
		Address:  p.address,
		LastSeen: p.lastSeen,
		Port:     p.port,
		Stats:    *p.stats,
	})
}

// UnmarshalJSON implements custom JSON deserialization for peer.
func (p *peer) UnmarshalJSON(data []byte) error {
	aux := struct {
		ID       string     `json:"id"`
		Address  string     `json:"address"`
		LastSeen time.Time  `json:"last_seen"`
		Port     int        `json:"port"`
		Stats    *PeerStats `json:"stats"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	p.id = aux.ID
	p.address = aux.Address
	p.lastSeen = aux.LastSeen
	p.port = aux.Port
	p.stats = aux.Stats

	return nil
}

// Peer (public) is the exported struct for returning peer data safely
type Peer struct {
	ID       string
	Address  string
	LastSeen time.Time
	Port     int
}

// PeerStore manages a list of discoverd peers securely
type PeerStore struct {
	mu      sync.RWMutex
	peers   map[string]*peer
	penalty time.Duration
}

// NewPeerStore initializes and returns a secure PeerStore
func NewPeerStore(penalty time.Duration) *PeerStore {
	return &PeerStore{
		peers:   make(map[string]*peer),
		penalty: penalty,
	}
}

// AddPeer safely adds a new peer into the store
func (ps *PeerStore) AddPeer(id, address string, port int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if p, exists := ps.peers[id]; exists {
		p.lastSeen = time.Now()
		p.stats.lastSeen = time.Now()
		p.updateStats(ps.penalty)
	} else {
		ps.peers[id] = &peer{
			id:       id,
			address:  address,
			lastSeen: time.Now(),
			port:     port,
			stats:    NewPeerStats(),
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
	now := time.Now()
	var toDelete []string

	ps.mu.RLock()
	for id, p := range ps.peers {

		if now.Sub(p.lastSeen) > expiry {
			toDelete = append(toDelete, id)
		}
	}
	ps.mu.RUnlock()

	if len(toDelete) > 0 {
		ps.mu.Lock()
		for _, id := range toDelete {
			delete(ps.peers, id)
		}
		ps.mu.Unlock()
	}
}

// updateStats updates stability scores for the peer
func (p *peer) updateStats(penalty time.Duration) {
	p.stats.computeStability(penalty)
}

// PeerStats stores stability information for a peer
type PeerStats struct {
	lastSeen       time.Time
	firstSeen      time.Time
	uptime         time.Duration
	dropCount      int
	stabilityScore float64
	// responseTime   time.Duration
}

// NewPeerStats initializes a new PeerStats entry
func NewPeerStats() *PeerStats {
	now := time.Now()
	return &PeerStats{
		lastSeen:  now,
		firstSeen: now,
		uptime:    0,
		dropCount: 0,
	}
}

// IsInactive checks if a peer has been inactive for too long
func (p *PeerStats) IsInactive(threshold time.Duration) bool {
	return time.Since(p.lastSeen) > threshold
}

// IncrementDropCount increases the drop count when a peer disappears
func (p *PeerStats) IncrementDropCount() {
	p.dropCount++
}

// computeStability will formulate and compute the reliability of a peer
func (ps *PeerStats) computeStability(penalty time.Duration) {
	ps.lastSeen = time.Now()
	ps.uptime = time.Since(ps.firstSeen)
	ps.stabilityScore = calculateStabilityScore(ps.uptime, penalty, ps.dropCount)
}

// Getters to allow controlled access
func (p *PeerStats) LastSeen() time.Time     { return p.lastSeen }
func (p *PeerStats) Uptime() time.Duration   { return p.uptime }
func (p *PeerStats) StabilityScore() float64 { return p.stabilityScore }
func (p *PeerStats) DropCount() int          { return p.dropCount }

func calculateStabilityScore(uptime, penalty time.Duration, dropCount int) float64 {
	total := uptime + (time.Duration(dropCount) * penalty)

	if total == 0 {
		return 1
	}
	return float64(uptime) / float64(total)
}
