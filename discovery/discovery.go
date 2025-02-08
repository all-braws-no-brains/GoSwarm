package discovery

import (
	"log"
	"time"
)

// PeerTracker manages discovered peers and their stability stats
type PeerTracker struct {
	peerStore    *PeerStore
	peerStorage  *PeerStorage
	stopChan     chan struct{}
	dumpInterval time.Duration
}

// NewPeerTracker initializes a peer tracker
func NewPeerTracker(peerStore *PeerStore, dbFile string, dumpInterval time.Duration) (*PeerTracker, error) {
	peerStorage, err := NewPeerStorage(dbFile)
	if err != nil {
		return nil, err
	}

	tracker := &PeerTracker{
		peerStore:    peerStore,
		peerStorage:  peerStorage,
		stopChan:     make(chan struct{}),
		dumpInterval: dumpInterval,
	}

	// load strored peers into memory
	peers, err := peerStorage.GetAllPeers()
	if err != nil {
		log.Println("Failed to load peers from storage: ", err)
	}

	peerStore.mu.Lock()
	for _, p := range peers {
		peerStore.peers[p.id] = p
	}
	peerStore.mu.Unlock()

	tracker.startPeriodicDump()
	return tracker, nil
}

func (pt *PeerTracker) startPeriodicDump() {
	go func() {
		ticker := time.NewTicker(pt.dumpInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pt.dumpPeersToDB()
			case <-pt.stopChan:
				pt.dumpPeersToDB()
				return
			}
		}
	}()
}

// dumpPeersToDB saves all in-memory peers to BoltDB
func (pt *PeerTracker) dumpPeersToDB() {
	pt.peerStore.mu.RLock()
	defer pt.peerStore.mu.RUnlock()

	for _, p := range pt.peerStore.peers {
		if err := pt.peerStorage.SavePeer(p); err != nil {
			log.Println("Failed to save peer:", err)
		}
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

	if err := pt.peerStorage.SavePeer(pt.peerStore.peers[id]); err != nil {
		log.Println("Failed to persist peer update: ", err)
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

			if err := pt.peerStorage.DeletePeer(id); err != nil {
				log.Println("Failed to delete peer from storage bucket: ", err)
			}
		}
	}
}

// StopCleanup stops the background cleanup process
func (pt *PeerTracker) StopCleanup() {
	close(pt.stopChan)
}

// Close ensures final data persistence
func (pt *PeerTracker) Close() {
	pt.dumpPeersToDB()
	pt.peerStorage.Close()
}
