package supervisor

import (
	"heis/types"
)

type PeerState int

const (
	PeerStateAlive PeerState = iota
	PeerStateSuspected
	PeerStateDead
)

// KUN FOR PRINTING
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

func formatPeerAliveness(update peerUpdate) types.PeerAliveness {
	return types.PeerAliveness{
		ID:      update.peerID,
		IsAlive: update.newState == PeerStateAlive,
	}
}
