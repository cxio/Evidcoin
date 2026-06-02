package services

// STUN is the interface boundary for the STUN/stun2p public service
// (第 15 章 §1，DEC-0603).
//
// STUN provides NAT traversal detection and hole-punching assistance to support
// direct UDP P2P connections between peers.
// Implementation is external (github.com/cxio/stun2p); this interface defines the
// boundary contract. STUN does not participate in blockchain consensus, block validation,
// transaction execution, PoH, or script verification.
// Service unavailability does not affect block legality.
type STUN interface {
	// Probe initiates a NAT traversal probe for the local node.
	// Returns the observed external address string, or an error if the probe fails.
	Probe() (externalAddr string, err error)

	// Assist requests hole-punching assistance for connecting to a remote peer.
	// peerAddr is the target peer address in host:port form.
	// Returns an error if the request fails.
	Assist(peerAddr string) error
}

// BaseNetwork is the interface boundary for the base P2P network (node discovery layer)
// (第 15 章 §1，DEC-0603).
//
// The base network provides a lightweight P2P node-discovery and general broadcast
// foundation that all service and application sub-networks rely on.
// Implementation is external (github.com/cxio/p2p); this interface defines the
// boundary contract. The base network does not participate in block/transaction/PoH/script
// verification. Service unavailability does not affect block legality.
type BaseNetwork interface {
	// Broadcast sends data to all connected peers in the local sub-network.
	// Returns an error if the broadcast cannot be initiated.
	Broadcast(data []byte) error

	// ConnectedPeers returns the count of currently connected peers.
	ConnectedPeers() int
}
