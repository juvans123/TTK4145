package supervisor
import (
	"context"
	"fmt"
	"elevator/types"
	"time"
)

type Supervisor struct {
	init                 SupervisorInit
	localTickCount       uint8
	outgoingHeartbeatsCh chan<- Heartbeat
	incomingHeartbeatsCh <-chan Heartbeat
	peerAlivenessCh      chan<- types.PeerAliveness
}

func New(
	init SupervisorInit,
	outgoingHeartbeatsCh chan<- Heartbeat,
	incomingHeartbeatsCh <-chan Heartbeat,
	peerAlivenessCh chan<- types.PeerAliveness,
) *Supervisor {
	return &Supervisor{
		init:                 init,
		localTickCount:       0,
		outgoingHeartbeatsCh: outgoingHeartbeatsCh,
		incomingHeartbeatsCh: incomingHeartbeatsCh,
		peerAlivenessCh:      peerAlivenessCh,
	}
}

func (s *Supervisor) MonitorPeerHealth(context context.Context) error {
	ticker := time.NewTicker(s.init.TickInterval)
	defer ticker.Stop()
	tracker := NewPeerTracker(
		s.init.TicksBeforeSuspected,
		s.init.ConsensusRequired,
	)

	fmt.Printf("[Supervisor %s] Starting\n", s.init.MyID)

	for {
		select {
		case <-context.Done():
			return context.Err()

		case <-ticker.C:
			s.localTickCount++
			s.sendHeartbeat(tracker)
			updates := s.detectAndConfirmPeerFailures(tracker)
			s.publishAlivenessTransitions(updates)

		case hb := <-s.incomingHeartbeatsCh:
			if hb.PeerID == s.init.MyID {
				break
			}
			updates := tracker.receivePeerHeartbeat(hb, s.init.MyID, s.localTickCount)
			s.publishAlivenessTransitions(updates)
			s.logStateTransitions(updates)
		}
	}
}

func (s *Supervisor) detectAndConfirmPeerFailures(tracker *PeerTracker) []peerUpdate {
	timeoutUpdates := tracker.markTimedOutPeersAsSuspected(s.localTickCount)
	deadUpdates := tracker.confirmDeadPeersIfConsensus()
	s.logStateTransitions(timeoutUpdates)
	s.logStateTransitions(deadUpdates)
	return append(timeoutUpdates, deadUpdates...)
}

func (s *Supervisor) sendHeartbeat(tracker *PeerTracker) {
	hb := Heartbeat{
		PeerID:         s.init.MyID,
		Counter:        s.localTickCount,
		SuspectedPeers: tracker.getNonAlivePeers(),
	}

	s.outgoingHeartbeatsCh <- hb

}

func (s *Supervisor) logStateTransitions(updates []peerUpdate) {
	for _, u := range updates {
		fmt.Printf("[Supervisor %s] %s: %s -> %s\n",
			s.init.MyID, u.peerID, u.oldState, u.newState)
	}
}

func (s *Supervisor) publishAlivenessTransitions(updates []peerUpdate) {
	for _, u := range updates {

		s.peerAlivenessCh <- formatPeerAliveness(u)
	}
}

func formatPeerAliveness(update peerUpdate) types.PeerAliveness {
	return types.PeerAliveness{
		ID:      update.peerID,
		IsAlive: update.newState == PeerStateAlive,
	}
}
