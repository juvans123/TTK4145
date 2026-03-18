package supervisor

import (
	"heis/types"
)


type PeerTracker struct {
	peers             map[string]*peerInfo
	suspectThreshold  int
	consensusRequired int
}


func NewPeerTracker(suspectThreshold, consensusRequired int) *PeerTracker {
	return &PeerTracker{
		peers:             make(map[string]*peerInfo),
		suspectThreshold:  suspectThreshold,
		consensusRequired: consensusRequired,
	}
}

func (pt *PeerTracker) countAliveRemotePeers() int {
	count := 0
	for _, peer := range pt.peers {
		if peer.state == PeerStateAlive {
			count++
		}
	}
	return count
}

func (pt *PeerTracker) hasDeathConsensusFor(peerID string) bool {
	peer, exists := pt.peers[peerID]
	if !exists {
		return false
	}

	suspicionCount := len(peer.suspectedBy)
	if peer.state == PeerStateSuspected {
		suspicionCount++
	}

	totalAliveNodes := pt.countAliveRemotePeers() + 1

	effectiveRequired := pt.consensusRequired
	if totalAliveNodes < effectiveRequired {
		effectiveRequired = totalAliveNodes
	}

	return suspicionCount >= effectiveRequired

}

func (pt *PeerTracker) markTimedOutPeersAsSuspected(currentTick uint8) []peerUpdate {
	var updates []peerUpdate
	for id, peer := range pt.peers {
		if peer.state == PeerStateDead {
			continue
		}
		missedHeartbeats := types.MissedTicksBetween(currentTick, peer.lastSeenAtTick)
		if missedHeartbeats >= pt.suspectThreshold && peer.state == PeerStateAlive {
			updates = append(updates, peerUpdate{
				peerID:   id,
				newState: PeerStateSuspected,
				oldState: PeerStateAlive,
			})
		}
	}
	pt.applyUpdates(updates)
	return updates
}

func (pt *PeerTracker) confirmDeadPeersIfConsensusReached() []peerUpdate {
	var updates []peerUpdate
	for id := range pt.peers {
		if u, ok := pt.buildDeathUpdateIfConsensusReached(id); ok {
			updates = append(updates, u)
		}
	}
	pt.applyUpdates(updates)
	return updates
}

func (pt *PeerTracker) receivePeerHeartbeat(hb Heartbeat, localNodeID string, localTick uint8) []peerUpdate {
	var updates []peerUpdate

	if u, ok := pt.recordHeartbeatFromSender(hb, localTick); ok {
		updates = append(updates, u)
	}

	updates = append(updates, pt.updateConsensus(hb, localNodeID)...)

	pt.applyUpdates(updates)

	return updates
}

func (pt *PeerTracker) recordHeartbeatFromSender(hb Heartbeat, localTick uint8) (peerUpdate, bool) {
	sender, exists := pt.peers[hb.PeerID]

	if !exists {
		return pt.registerPeer(hb, localTick), true
	}

	if sender.state == PeerStateDead {
		return pt.rejoinPeer(hb, sender, localTick), true
	}

	if types.IsSequentiallyNewer(hb.Counter, sender.lastReceivedCounter) {
		return pt.refreshPeerCounters(hb, sender, localTick)
	}

	return peerUpdate{}, false
}

func (pt *PeerTracker) registerPeer(hb Heartbeat, localTicker uint8) peerUpdate {
	info := &peerInfo{}
	resetPeerToAlive(info, hb, localTicker)
	pt.peers[hb.PeerID] = info
	return newAliveUpdate(hb.PeerID, PeerStateDead)
}

func (pt *PeerTracker) rejoinPeer(hb Heartbeat, sender *peerInfo, localTicker uint8) peerUpdate {
	oldState := sender.state
	resetPeerToAlive(sender, hb, localTicker)
	return newAliveUpdate(hb.PeerID, oldState)
}

func resetPeerToAlive(info *peerInfo, hb Heartbeat, localTicker uint8) {
	info.lastReceivedCounter = hb.Counter
	info.lastSeenAtTick = localTicker
	info.state = PeerStateAlive
	info.suspectedBy = make(map[string]bool)
}

func (pt *PeerTracker) refreshPeerCounters(hb Heartbeat, sender *peerInfo, localTick uint8) (peerUpdate, bool) {
	oldState := sender.state
	sender.lastReceivedCounter = hb.Counter
	sender.lastSeenAtTick = localTick

	if oldState == PeerStateSuspected {
		sender.state = PeerStateAlive
		sender.suspectedBy = make(map[string]bool)
		return newAliveUpdate(hb.PeerID, oldState), true
	}

	return peerUpdate{}, false
}

func (pt *PeerTracker) updateConsensus(hb Heartbeat, myID string) []peerUpdate {
	var updates []peerUpdate

	for _, suspectedID := range hb.SuspectedPeers {
		if suspectedID == myID {
			continue
		}
		if u, ok := pt.addSuspicionAndCheckDeath(suspectedID, hb.PeerID); ok {
			updates = append(updates, u)
		}
	}

	return updates
}

func (pt *PeerTracker) buildDeathUpdateIfConsensusReached(peerID string) (peerUpdate, bool) {
	peer, exists := pt.peers[peerID]
	if !exists || peer.state != PeerStateSuspected {
		return peerUpdate{}, false
	}
	if !pt.hasDeathConsensusFor(peerID) {
		return peerUpdate{}, false
	}
	return peerUpdate{
		peerID:   peerID,
		newState: PeerStateDead,
		oldState: PeerStateSuspected,
	}, true
}

func (pt *PeerTracker) addSuspicionAndCheckDeath(suspectedID, reporterID string) (peerUpdate, bool) {
	peer, exists := pt.peers[suspectedID]
	if !exists {
		return peerUpdate{}, false
	}

	peer.suspectedBy[reporterID] = true
	return pt.buildDeathUpdateIfConsensusReached(suspectedID)
}

func (pt *PeerTracker) applyUpdates(updates []peerUpdate) {
	for _, u := range updates {
		peer, exists := pt.peers[u.peerID]
		if !exists {
			continue
		}

		peer.state = u.newState

		if u.newState == PeerStateAlive && u.oldState != PeerStateAlive {
			peer.suspectedBy = make(map[string]bool)
		}

		pt.peers[u.peerID] = peer
	}
}

func (pt *PeerTracker) getNonAlivePeers() []string {
	var suspected []string
	for id, peer := range pt.peers {
		if peer.state == PeerStateSuspected || peer.state == PeerStateDead {
			suspected = append(suspected, id)
		}
	}
	return suspected
}