package discovery

import (
	"os"
	"sync"
	"testing"
	"time"
)

var testPeer = &peer{id: "peer_1", address: "192.168.1.1", port: 8080, lastSeen: time.Now(), stats: NewPeerStats()}

func setupTestStorage(t *testing.T) (*PeerStorage, func()) {
	dbFile := "test_peers.db"
	ps, err := NewPeerStorage(dbFile)
	if err != nil {
		t.Fatalf("Failed to initialize PeerStorage: %v", err)
	}

	cleanUp := func() {
		ps.Close()
		os.Remove(dbFile)
	}

	return ps, cleanUp
}

func TestSaveAndGetPeer(t *testing.T) {
	ps, cleanup := setupTestStorage(t)
	defer cleanup()

	p := testPeer
	err := ps.SavePeer(p)
	if err != nil {
		t.Fatalf("Failed to save peer: %v", err)
	}

	retrieved, err := ps.GetPeer("peer_1")
	if err != nil {
		t.Fatalf("Failed tp retrieve peer: %v", err)
	}

	if retrieved.id != p.id || retrieved.address != p.address || retrieved.port != p.port {
		t.Errorf("Retrieved peer data mismatch. Got %+v, expected %+v", retrieved, p)
	}
}

func TestDeletePeer(t *testing.T) {
	ps, cleanUp := setupTestStorage(t)

	defer cleanUp()

	p := testPeer
	ps.SavePeer(p)

	err := ps.DeletePeer("peer_1")
	if err != nil {
		t.Fatalf("Failed to delete peer: %v", err)
	}

	_, err = ps.GetPeer("peer_1")
	if err == nil {
		t.Errorf("Expected error when retrieving deleted peer, but got none")
	}
}

func TestConcurrentAccess(t *testing.T) {
	ps, cleanUp := setupTestStorage(t)
	defer cleanUp()

	var wg sync.WaitGroup
	peerCount := 100

	// concurrenty store peers
	for i := 0; i < peerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			p := &peer{id: string(rune(id)), address: "192.168.1.1", port: 8080, stats: NewPeerStats()}
			if err := ps.SavePeer(p); err != nil {
				t.Errorf("Failed to save peer %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// concurrently retrieve peers
	for i := 0; i < peerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := ps.GetPeer(string(rune(id)))
			if err != nil {
				t.Errorf("Failed to get peer %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// concurrently delete peers
	for i := 0; i < peerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := ps.DeletePeer(string(rune(id))); err != nil {
				t.Errorf("Failed to delete peer %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
}
