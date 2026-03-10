package network

// NY FIL: wrapper for Heartbeat-sending og -mottak.
// Tidligere sendte supervisor direkte på rå nettverkskanaler (hbTx/hbRx)
// som ble koblet til bcast.Transmitter/Receiver i main.go.
// Nå er supervisor isolert fra nettverkslaget.

import (
	"heis/supervisor"
)

func RunHeartbeatBroadcast(
	in <- chan supervisor.Heartbeat,
	netTX chan <- supervisor.Heartbeat, 
) {
	for hb := range in {
		select {
		case netTX <- hb:
		default:
		}
	}
}

func RunHeartbeatReceive(
	MyID string, 
	netRx <- chan supervisor.Heartbeat, 
	out chan <- supervisor.Heartbeat, 
) {
	for hb := range netRx {
		if hb.PeerID == MyID {
			continue
		}
		select{
		case out <- hb: 
		default:
		}
	}
}