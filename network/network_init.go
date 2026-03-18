package network

type NetworkPorts struct {
	StatePort     int
	OrderPort     int
	HeartbeatPort int
}

func DefaultNetworkPorts() NetworkPorts {
	return NetworkPorts{
		StatePort:     16570,
		OrderPort:     16571,
		HeartbeatPort: 16647,
	}
}