package supervisor

import (
	"context"
	"fmt"
	"time"
)

type Supervisor struct {
	config    Config
	myCounter uint8
	hbTx         chan<- Heartbeat
	hbRx         <-chan Heartbeat
	peerUpdateTx chan<- peerUpdate
	PeerEventTx  chan<- PeerEvent

	/*Fuck is this */
	stateCh chan chan map[string]PeerState //Finn ut
}

/*Fuck is this */
func New(
	cfg Config,
	hbTx chan<- Heartbeat,
	hbRx <-chan Heartbeat,
	PeerEventTx chan<- PeerEvent,

) *Supervisor {
	return &Supervisor{
		config:       cfg,
		myCounter:    0,
		hbTx:         hbTx,
		hbRx:         hbRx,
		stateCh:      make(chan chan map[string]PeerState),
	}
}	

func (s *Supervisor) MonitorPeerHealth(ctx context.Context) error {
	ticker := time.NewTicker(s.config.TickInterval)
	defer ticker.Stop()
	tracker := NewPeerTracker(
		s.config.SuspectThreshold,
		s.config.DeadThreshold,
		s.config.ConsensusRequired,
	)

	fmt.Printf("[Supervisor %s] Starting\n", s.config.MyID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			s.myCounter++
			hb := Heartbeat{
				PeerID:         s.config.MyID,
				Counter:        s.myCounter,
				SuspectedPeers: tracker.getSuspectedPeers(),
			}

			select {
			case s.hbTx <- hb:
			default:
			}
			s.sendEvents(tracker.detectHeartbeatTimeouts(s.myCounter))

		case hb := <-s.hbRx:
			if hb.PeerID == s.config.MyID {
				continue
			}
			s.sendEvents(tracker.processHeartbeat(hb, s.config.MyID, s.myCounter))
		}
	}
}

func (s *Supervisor) sendEvents(updates []peerUpdate) {
	for _, u := range updates {
		fmt.Printf("[Supervisor %s] %s: %s -> %s\n",
			s.config.MyID, u.peerID, u.oldState, u.newState)
	}
}