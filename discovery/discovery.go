package discovery

import (
	"time"
)

// PeerTracker manages discovered peers and their stability stats
type PeerTracker struct {
	peerStore *PeerStore
	stopChan  chan struct{}
}

// NewPeerTracker initializes a peer tracker
func NewPeerTracker(peerStore *PeerStore) *PeerTracker {
	return &PeerTracker{
		peerStore: peerStore,
		stopChan:  make(chan struct{}),
	}
}

// UpdatePeer updates a peer's uptime and last seen time
func (pt *PeerTracker) UpdatePeer(id, address string, port int) {
	pt.peerStore.mu.Lock()
	defer pt.peerStore.mu.Unlock()

	if p, exists := pt.peerStore.peers[id]; exists {
		p.lastSeen = time.Now()
		p.stats.lastSeen = time.Now()
		p.updateStats(pt.peerStore.penalty)
	} else {
		pt.peerStore.peers[id] = &peer{
			id:       id,
			address:  address,
			lastSeen: time.Now(),
			stats:    NewPeerStats(),
			port:     port,
		}
	}
}

// StartCleanup runs a periodic cleanup process to remove inactive peers
func (pt *PeerTracker) StartCleanup(threshold time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				select {
				case <-pt.stopChan:
					return
				default:
					pt.cleanupPeers(threshold)
				}
			case <-pt.stopChan:
				return
			}
		}
	}()
}

// cleanupPeers removes peers that have been inactive for too long
func (pt *PeerTracker) cleanupPeers(threshold time.Duration) {
	pt.peerStore.mu.Lock()
	defer pt.peerStore.mu.Unlock()

	for id, peer := range pt.peerStore.peers {
		if peer.stats.IsInactive(threshold) {
			peer.stats.IncrementDropCount()
			delete(pt.peerStore.peers, id)
		}
	}
}

// StopCleanup stops the background cleanup process
func (pt *PeerTracker) StopCleanup() {
	close(pt.stopChan)
}
