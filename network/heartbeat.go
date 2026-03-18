package network

import (
	"heis/supervisor"
)

func ForwardOutgoingHeartbeats(
	in <-chan supervisor.Heartbeat,
	netTX chan<- supervisor.Heartbeat,
) {
	for hb := range in {
		select {
		case netTX <- hb:
		default:
		}
	}
}

func DeliverIncomingHeartbeats(
	myID string,
	netRx <-chan supervisor.Heartbeat,
	out chan<- supervisor.Heartbeat,
) {
	for hb := range netRx {
		if hb.PeerID == myID {
			continue
		}
		out <- hb
	}
}
