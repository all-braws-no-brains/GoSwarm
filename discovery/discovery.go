package discovery

import (
	"sync"
	"time"
)

// PeerTracker manages discovered peers and their stability stats
type PeerTracker struct {
	peerStore *PeerStore
	mu        sync.Mutex
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
func (pt *PeerTracker) UpdatePeer(id string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.peerStore.mu.Lock()
	defer pt.peerStore.mu.Unlock()

	if p, exists := pt.peerStore.peers[id]; exists {
		p.lastSeen = time.Now()
		p.updateStats(pt.peerStore.penalty)
	} else {
		pt.peerStore.peers[id] = &peer{
			id:       id,
			lastSeen: time.Now(),
			stats:    NewPeerStats(),
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
				pt.cleanupPeers(threshold)
			case <-pt.stopChan:
				return
			}
		}
	}()
}

// cleanupPeers removes peers that have been inactive for too long
func (pt *PeerTracker) cleanupPeers(threshold time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

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
