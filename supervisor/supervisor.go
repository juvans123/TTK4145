package supervisor

import (
	"context"
	"fmt"
	"heis/config"
	"time"
)

type Supervisor struct {
	config      Config
	myCounter   uint8
	hbTx        chan<- Heartbeat
	hbRx        <-chan Heartbeat
	PeerEventTx chan<- config.PeerEvent
}

func New(
	cfg Config,
	hbTx chan<- Heartbeat,
	hbRx <-chan Heartbeat,
	peerEventTx chan<- config.PeerEvent,
) *Supervisor {
	return &Supervisor{
		config:      cfg,
		myCounter:   0,
		hbTx:        hbTx,
		hbRx:        hbRx,
		PeerEventTx: peerEventTx,
	}
}

func (s *Supervisor) MonitorPeerHealth(ctx context.Context) error {
	ticker := time.NewTicker(s.config.SupervisorConfig.TickInterval)
	defer ticker.Stop()
	tracker := NewPeerTracker(
		s.config.SupervisorConfig.SuspectThreshold,
		s.config.SupervisorConfig.ConsensusRequired,
	)

	fmt.Printf("[Supervisor %s] Starting\n", s.config.MyID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.handleTick(tracker)
		case hb := <-s.hbRx:
			s.handleIncomingHeartbeat(hb, tracker)
		}
	}
}

func (s *Supervisor) handleTick(tracker *peerTracker) {
	s.myCounter++
	s.sendHeartbeat(tracker)
	updates := tracker.detectHeartbeatTimeouts(s.myCounter)
	s.sendEvents(updates)
}

func (s *Supervisor) sendHeartbeat(tracker *peerTracker) {
	hb := Heartbeat{
		PeerID:         s.config.MyID,
		Counter:        s.myCounter,
		SuspectedPeers: tracker.getSuspectedPeers(),
	}
	select {
	case s.hbTx <- hb:
	default:
	}
}

func (s *Supervisor) handleIncomingHeartbeat(hb Heartbeat, tracker *peerTracker) {
	if hb.PeerID == s.config.MyID {
		return
	}
	updates := tracker.processHeartbeat(hb, s.config.MyID, s.myCounter)
	s.sendEvents(updates)
}

func (s *Supervisor) sendEvents(updates []peerUpdate) {
	for _, u := range updates {
		fmt.Printf("[Supervisor %s] %s: %s -> %s\n",
			s.config.MyID, u.peerID, u.oldState, u.newState)

		if u.newState == SuspectedDead{ //Vi hopper over å sende suspecteddead
			continue
		}

		select {
		case s.PeerEventTx <- toPeerEvent(u):
		default:
		}
	}
}

func toPeerEvent(u peerUpdate) config.PeerEvent {
	return config.PeerEvent{
		PeerID: u.peerID,
		Alive:  u.newState == Alive,
	}
}