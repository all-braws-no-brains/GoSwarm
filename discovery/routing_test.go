package discovery

import "testing"

func TestPeerRoutingTable(t *testing.T) {
	rt := newRoutingTable()

	rt.addPeer("peer_1", "192.168.1.10.5000")
	rt.addPeer("peer_2", "192.168.1.10.5001")

	addr, exists := rt.getPeer("peer_1")
	if !exists || addr != "192.168.1.10.5000" {
		t.Errorf("Expected peer_1 address to be 192.168.1.10.5000, got %s", addr)
	}

	addr, exists = rt.getPeer("peer_2")
	if !exists || addr != "192.168.1.10.5001" {
		t.Errorf("Expected peer_2 address to be 192.168.1.10.5001, got %s", addr)
	}

	rt.removePeer("peer_1")
	_, exists = rt.getPeer("peer_1")
	if exists {
		t.Errorf("peer_1 should be removed but still exists")
	}
}

func TestConcurrentPeerRoutingTable(t *testing.T) {
	rt := newRoutingTable()
	peerCount := 30000
	done := make(chan bool)

	for i := 0; i < peerCount; i++ {
		go func(id int) {
			rt.addPeer(string(rune(id)), "192.168.1.1.5000")
			done <- true
		}(i)
	}

	for i := 0; i < peerCount; i++ {
		<-done
	}

	if len(rt.getAllPeers()) != peerCount {
		t.Errorf("Expected %d peers, but found %d", peerCount, len(rt.getAllPeers()))
	}
}
