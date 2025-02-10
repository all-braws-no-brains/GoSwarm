# GoSwarm

GoSwarm is a **lightweight and modular P2P mesh networking library** written in **Go**, designed to enable **direct device-to-device communication** without relying on a central server. The library provides **peer discovery, persistence, and TCP-based communication**, allowing developers to build custom distributed applications.

## Features Provided

✅ **Peer Discovery**  
- Uses **mDNS (Multicast DNS)** for decentralized peer discovery.  
- Nodes broadcast their presence and listen for other peers on the network.  

✅ **Persistent Peer Storage**  
- Uses **BoltDB** for storing known peers across restarts.  
- Peers are saved with metadata like last seen time and stability score.  
- Implements automatic cleanup for inactive peers.  

✅ **TCP-Based Peer Communication**  
- Supports **bi-directional communication** over TCP.  
- Allows users to **send and receive arbitrary data** (agnostic to message type).  
- Provides a simple **event-driven API** for handling messages.  

✅ **Concurrency-Safe Design**  
- Uses **mutexes (sync.Mutex, sync.RWMutex)** for thread-safe operations.  
- Optimized for high-concurrency environments.  

---

## Installation

```sh
go get github.com/all-braws-no-brains/goswarm