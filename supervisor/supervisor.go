package supervisor

// MEGET KRITISK MED NON-BLOCKING SEND
// VURDER HANDLEINCOMING HEARTBEATS

import (
	"context"
	"fmt"
	"heis/config"
	"time"
)

type Supervisor struct {
	config               Config
	localTickCount       uint8
	outgoingHeartbeatsCh chan<- Heartbeat
	incomingHeartbeatsCh <-chan Heartbeat
	PeerEventTx          chan<- config.PeerEvent
}

func New(
	cfg Config,
	outgoingHeartbeatsCh chan<- Heartbeat,
	incomingHeartbeatsCh <-chan Heartbeat,
	peerEventTx chan<- config.PeerEvent,
) *Supervisor {
	return &Supervisor{
		config:               cfg,
		localTickCount:       0,
		outgoingHeartbeatsCh: outgoingHeartbeatsCh,
		incomingHeartbeatsCh: incomingHeartbeatsCh,
		PeerEventTx:          peerEventTx,
	}
}

func (s *Supervisor) MonitorPeerHealth(ctx context.Context) error {
	ticker := time.NewTicker(s.config.tickInterval())
	defer ticker.Stop()
	tracker := NewPeerTracker(
		s.config.suspectThreshold(),
		s.config.consensusRequired(),
	)

	fmt.Printf("[Supervisor %s] Starting\n", s.config.MyID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.broadcastHeartbeatAndDetectTimeouts(tracker)
		case hb := <-s.incomingHeartbeatsCh:
			s.handleIncomingHeartbeat(hb, tracker)
		}
	}
}

func (s *Supervisor) broadcastHeartbeatAndDetectTimeouts(tracker *PeerTracker) {
	s.localTickCount++
	s.sendHeartbeat(tracker)

	susUpdate:=tracker.markTimedOutPeersAsSuspected(s.localTickCount)
	s.logStateTransitions(susUpdate)


	deadUpdates := tracker.confirmDeadPeersIfConsensusReached()
	s.logStateTransitions(deadUpdates)
	s.notifyOrderManagerOfConfirmedTransitions(deadUpdates)
}

func (s *Supervisor) sendHeartbeat(tracker *PeerTracker) {
	hb := Heartbeat{
		PeerID:         s.config.MyID,
		Counter:        s.localTickCount,
		SuspectedPeers: tracker.getSuspectedPeers(),
	}
	select {
	case s.outgoingHeartbeatsCh <- hb:
	default:
	}
}

func (s *Supervisor) handleIncomingHeartbeat(hb Heartbeat, tracker *PeerTracker) {
	if hb.PeerID == s.config.MyID {
		return
	}
	updates := tracker.receivePeerHeartbeat(hb, s.config.MyID, s.localTickCount)
	s.notifyOrderManagerOfConfirmedTransitions(updates)
	s.logStateTransitions(updates)

}

func (s *Supervisor) logStateTransitions(updates []peerUpdate) {
	for _, u := range updates {
		fmt.Printf("[Supervisor %s] %s: %s -> %s\n",
			s.config.MyID, u.peerID, u.oldState, u.newState)
	}
}

func (s *Supervisor) notifyOrderManagerOfConfirmedTransitions(updates []peerUpdate) {
	for _, u := range updates {
		select {
		case s.PeerEventTx <- toPeerEvent(u):
		default:
		}
	}
}

func toPeerEvent(u peerUpdate) config.PeerEvent {
	return config.PeerEvent{
		PeerID: u.peerID,
		Alive:  u.newState == PeerStateAlive,
	}
}