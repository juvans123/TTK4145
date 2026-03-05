
package supervisor

import "heis/config"

type PeerState int

const (
	Alive PeerState = iota
	SuspectedDead
	Dead
)

func (ps PeerState) String() string {
	switch ps {
	case Alive:
		return "Alive"
	case SuspectedDead:
		return "SuspectedDead"
	case Dead:
		return "Dead"
	default:
		return "Unknown"
	}
}

type Heartbeat struct {
	PeerID         string
	Counter        uint8
	SuspectedPeers []string
}

// Config bruker heis/config.SupervisorConfig som kilde
type Config struct {
	MyID             string
	SupervisorConfig config.SupervisorConfig
}

type PeerEvent struct {
	PeerID string
	Alive  bool
}

func NewConfig(myID string) Config {
	return Config{
		MyID:             myID,
		SupervisorConfig: config.DefaultSupervisorConfig(),
	}
}

// Hjelpere for å lese konfig-verdier
func (c Config) tickInterval() interface{} { return c.SupervisorConfig.TickInterval }
func (c Config) suspectThreshold() int     { return c.SupervisorConfig.SuspectThreshold }
func (c Config) consensusRequired() int    { return c.SupervisorConfig.ConsensusRequired }