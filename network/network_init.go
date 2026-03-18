package network

type NetworkConfig struct {
	StatePort     int
	OrderPort     int
	HeartbeatPort int
}

func DefaultNetworkConfig() NetworkConfig {
	return NetworkConfig{
		StatePort:     16570,
		OrderPort:     16571,
		HeartbeatPort: 16647,
	}
}