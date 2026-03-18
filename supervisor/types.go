package supervisor

import (
	"heis/config"
	"time"
)

type PeerState int

const (
	PeerStateAlive PeerState = iota
	PeerStateSuspected
	PeerStateDead
)
//KUN FOR PRINTING
func (ps PeerState) String() string {
	switch ps {
	case PeerStateAlive:
		return "Alive"
	case PeerStateSuspected:
		return "SuspectedDead"
	case PeerStateDead:
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

type Config struct {
	MyID             string
	SupervisorConfig config.SupervisorConfig
}

func NewConfig(myID string) Config {
	return Config{
		MyID:             myID,
		SupervisorConfig: config.DefaultSupervisorConfig(),
	}
}

type peerUpdate struct {
	peerID   string
	newState PeerState
	oldState PeerState
}

type peerInfo struct {
	lastReceivedCounter uint8
	lastSeenAtTick      uint8 
	state               PeerState
	suspectedBy         map[string]bool
}


func newAliveUpdate(peerID string, oldState PeerState) peerUpdate {
	return peerUpdate{
		peerID:   peerID,
		newState: PeerStateAlive,
		oldState: oldState,
	}
}

func toPeerEvent(u peerUpdate) config.PeerAliveness {
	return config.PeerAliveness{
		ID: u.peerID,
		IsAlive:  u.newState == PeerStateAlive,
	}
}

// Hjelpere for å lese konfig-verdier
func (c Config) tickInterval() time.Duration { return c.SupervisorConfig.TickInterval }
func (c Config) suspectThreshold() int     { return c.SupervisorConfig.SuspectThreshold }
func (c Config) consensusRequired() int    { return c.SupervisorConfig.ConsensusRequired }