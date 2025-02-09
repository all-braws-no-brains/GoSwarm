package discovery

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewPeerStore(t *testing.T) {
	penalty := 10 * time.Second
	ps := NewPeerStore(penalty)

	if ps == nil {
		t.Fatal("Expected PeerStore to be intialized, got nil")
	}

	if len(ps.GetPeers()) != 0 {
		t.Errorf("Expected empty peer list, got %d peers", len(ps.GetPeers()))
	}
}

func TestAddPeer(t *testing.T) {
	ps := NewPeerStore(10 * time.Second)

	ps.AddPeer("peer_1", "192.168.1.1", 8080)
	ps.AddPeer("peer_2", "192.168.1.2", 9090)

	peers := ps.GetPeers()
	if len(peers) != 2 {
		t.Errorf("Expected 2 peers, got %d", len(peers))
	}
}

func TestAddPeerConcurrent(t *testing.T) {
	ps := NewPeerStore(10 * time.Second)

	var wg sync.WaitGroup
	peerCount := 1000

	for i := 0; i < peerCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ps.AddPeer(fmt.Sprintf("peer_%d", i), fmt.Sprintf("192.168.1.%d", i), 8000+i)
		}(i)
	}

	wg.Wait()

	peers := ps.GetPeers()
	if len(peers) != peerCount {
		t.Errorf("Expected %d peers, got %d", peerCount, len(peers))
	}
}

func TestUpdatePeer(t *testing.T) {
	ps := NewPeerStore(10 * time.Second)

	ps.AddPeer("peer_1", "192.168.1.1", 8080)
	time.Sleep(50 * time.Millisecond)
	ps.AddPeer("peer_1", "192.168.1.1", 8080)

	peers := ps.GetPeers()
	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}

	if peers[0].LastSeen.Before(time.Now().Add(-1 * time.Second)) {
		t.Errorf("Expected LastSeen to be updated, but it wasnt")
	}
}

func TestCleanupPeers(t *testing.T) {
	ps := NewPeerStore(100 * time.Millisecond)

	ps.AddPeer("peer_1", "192.168.1.1", 8080)
	ps.AddPeer("peer_2", "192.168.1.2", 9090)

	time.Sleep(200 * time.Millisecond)
	ps.Cleanup(100 * time.Millisecond)

	peers := ps.GetPeers()
	if len(peers) != 0 {
		t.Errorf("Expected all peers to be cleaned up, but got %d remaining", len(peers))
	}
}

func TestCleanupConcurrently(t *testing.T) {
	ps := NewPeerStore(100 * time.Millisecond)

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	// A routine to continuously add peers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			select {
			case <-stopChan:
				return
			default:
				ps.AddPeer(fmt.Sprintf("peer_%d", i), fmt.Sprintf("192.168.1.%d", i), 8000+i)
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	// A routine to cleanup in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			ps.Cleanup(50 * time.Millisecond)
			time.Sleep(20 * time.Millisecond)
		}
	}()

	// Allow both routines to run for a while, then stop
	time.Sleep(500 * time.Millisecond)
	close(stopChan)

	wg.Wait()

	// Check if any peers are left after cleanup
	peers := ps.GetPeers()
	t.Logf("Remaining peers after cleanup: %d", len(peers))
}
