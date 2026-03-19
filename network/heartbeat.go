package network

import (
	"elevator/supervisor"
)

func ForwardOutgoingHeartbeats(
	hbFromSupervisor <-chan supervisor.Heartbeat,
	netTX chan<- supervisor.Heartbeat,
) {
	for hb := range hbFromSupervisor {
		select {
		case netTX <- hb:
		default:
		}
	}
}

func DeliverIncomingHeartbeats(
	myID string,
	netRx <-chan supervisor.Heartbeat,
	hbToSupervisor chan<- supervisor.Heartbeat,
) {
	for hb := range netRx {
		if hb.PeerID == myID {
			continue
		}
		hbToSupervisor <- hb
	}
}
