package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/grandcat/zeroconf"
)

// mDNSService handles peer discovery using mDNS
type mDNSService struct {
	serviceName string
	instance    *zeroconf.Server
	resolver    *zeroconf.Resolver
	entries     chan *zeroconf.ServiceEntry

	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
	once    sync.Once
}

// NewMDNSService creates a new mDNS service instance
func NewMDNSService(serviceName string) *mDNSService {
	return &mDNSService{
		serviceName: serviceName,
	}
}

// Start registers current peer on the mDNS network
func (m *mDNSService) Start(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.instance != nil {
		return fmt.Errorf("service already running")
	}

	host, err := getLocalIP()
	if err != nil {
		return fmt.Errorf("failed to get local IP: %w", err)
	}

	instance, err := zeroconf.Register(
		"GoSwarmPeer",            // Instance Name
		m.serviceName,            // Service Type
		"local",                  // Domain
		port,                     // Port Number
		[]string{"GoSwarm Node"}, // TXT Records
		nil,                      // No Custom Interface
	)

	if err != nil {
		return fmt.Errorf("failed to register mDNS: %w", err)
	}

	m.instance = instance
	log.Printf("[mDNS] Registered service at %s:%d\n", host, port)
	return nil
}

// Stop shuts down the mDNS service
func (m *mDNSService) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.instance != nil {
		m.instance.Shutdown()
		log.Println("[mDNS] Service stopped")
		m.instance = nil
	}

	m.StopDiscovery()
}

// StopDiscovery stops the ongoing peer discovery
func (m *mDNSService) StopDiscovery() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	m.once.Do(func() {
		if m.entries != nil {
			close(m.entries)
			m.entries = nil
		}
	})

	m.running = false
	log.Println("[mDNS] Discovery stopped")
}

// PauseDiscovery pauses peer discovery without closing channels
func (m *mDNSService) PauseDiscovery() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.running = false
	log.Println("[mDNS] Discovery paused")
}

func (m *mDNSService) StartDiscover(foundPeer func(ip string, port int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("discovery already running")
	}

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalf("[mDNS] Failed to initialize resolver: %v", err)
	}
	m.resolver = resolver

	// initializing channel only when discovery starts
	m.entries = make(chan *zeroconf.ServiceEntry)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.running = true

	// routine to continuously discover peers
	go func() {
		for entry := range m.entries {
			if len(entry.AddrIPv4) > 0 {
				ip := entry.AddrIPv4[0].String()
				port := entry.Port
				foundPeer(ip, port)
			}
		}
	}()

	// start browsing in a separate goroutine
	go func() {
		err = resolver.Browse(ctx, m.serviceName, "local", m.entries)
		if err != nil {
			log.Fatalf("[mDNS] Failed to browse: %v", err)
		}
	}()

	log.Println("[mDNS] Discovery started")
	return nil
}

// getLocalIP retrieves the local machine's IP address
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no valid local IP found")
}
