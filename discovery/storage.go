package discovery

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

type PeerStorage struct {
	db   *bolt.DB
	mu   sync.RWMutex
	once sync.Once
}

// Constant for the storage bucket name
const peerBucket = "peers"

// PeerStorage initiliazes BoltDB for peer storage
func NewPeerStorage(dbFile string) (*PeerStorage, error) {
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	// ensure bucket exists
	if err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(peerBucket))
		return err
	}); err != nil {
		db.Close()
		return nil, err
	}

	return &PeerStorage{db: db}, nil
}

// SavePeer stores a peer in BoltDB
func (ps *PeerStorage) SavePeer(peer *peer) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	peer.lastSeen = time.Now()
	data, err := peer.MarshalJSON()
	if err != nil {
		return err
	}

	return ps.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(peerBucket))
		if bucket == nil {
			return errors.New("peer storage bucket not found")
		}
		return bucket.Put([]byte(peer.id), data)
	})
}

// GetPeers loads all peers form BoltDB
func (ps *PeerStorage) GetAllPeers() ([]*peer, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var peers []*peer
	if err := ps.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(peerBucket))
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(_, data []byte) error {
			var p peer
			if err := p.UnmarshalJSON(data); err != nil {
				return err
			}

			peers = append(peers, &p)
			return nil
		})
	}); err != nil {
		return nil, err
	}

	return peers, nil
}

// GetPeer retreives peer by id from BoltDB
func (ps *PeerStorage) GetPeer(id string) (*peer, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var p peer
	if err := ps.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(peerBucket))
		if bucket == nil {
			return errors.New("peer storage bucket not found")
		}
		data := bucket.Get([]byte(id))
		if data == nil {
			return errors.New("peer not found")
		}
		return p.UnmarshalJSON(data)
	}); err != nil {
		return nil, err
	}

	return &p, nil
}

// DeletePeer removes a peer from BoltDB
func (ps *PeerStorage) DeletePeer(id string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	return ps.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(peerBucket))
		if b == nil {
			return errors.New("peer storage bucket not found")
		}
		return b.Delete([]byte(id))
	})
}

// Close closes the database connection
func (ps *PeerStorage) Close() {
	ps.once.Do(func() {
		ps.mu.Lock()
		defer ps.mu.Unlock()

		if ps.db != nil {
			if err := ps.db.Close(); err != nil {
				log.Println("Error closing database:", err)
			}
			ps.db = nil
		}
	})
}

// CleanupPeers removes peers that haven't been seen for a set duration
func (ps *PeerStorage) CleanupPeers(expiryTime time.Duration) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	return ps.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(peerBucket))
		if bucket == nil {
			return errors.New("peer storage bucket not found")
		}
		return bucket.ForEach(func(k, v []byte) error {
			var p peer
			if err := p.UnmarshalJSON(v); err != nil {
				return err
			}
			if time.Since(p.lastSeen) > expiryTime {
				return bucket.Delete(k)
			}
			return nil
		})
	})
}
