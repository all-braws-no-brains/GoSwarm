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
type MDNSService struct {
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
func NewMDNSService(serviceName string) *MDNSService {
	return &MDNSService{
		serviceName: serviceName,
	}
}

// Start registers current peer on the mDNS network
func (m *MDNSService) Start(port int) error {
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
func (m *MDNSService) Stop() {
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
func (m *MDNSService) StopDiscovery() {
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
func (m *MDNSService) PauseDiscovery() {
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

func (m *MDNSService) StartDiscover(foundPeer func(ip string, port int)) error {
	m.mu.Lock()

	if m.running {
		m.mu.Unlock()
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
	m.mu.Unlock()

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
			log.Println("[mDNS] Failed to browse: ", err)
		}
	}()

	log.Println("[mDNS] Discovery started")
	return nil
}

// getLocalIP retrieves the local machine's IP address
func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", nil
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue // Skip down interfaces and loopbacks
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil // return the first valid IP found
			}
		}
	}

	return "", fmt.Errorf("no valid local IP found")
}
