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

func (peerTracker *PeerTracker) countAliveRemotePeers() int {
	count := 0
	for _, peer := range peerTracker.peers {
		if peer.state == PeerStateAlive {
			count++
		}
	}
	return count
}

func (peerTracker *PeerTracker) hasDeathConsensusFor(peerID string) bool {
	peer, exists := peerTracker.peers[peerID]
	if !exists {
		return false
	}

	suspicionCount := len(peer.suspectedBy)
	if peer.state == PeerStateSuspected {
		suspicionCount++
	}

	totalAliveNodes := peerTracker.countAliveRemotePeers() + 1

	effectiveRequired := peerTracker.consensusRequired
	if totalAliveNodes < effectiveRequired {
		effectiveRequired = totalAliveNodes
	}

	return suspicionCount >= effectiveRequired

}

func (peerTracker *PeerTracker) markTimedOutPeersAsSuspected(currentTick uint8) []peerUpdate {
	var updates []peerUpdate
	for id, peer := range peerTracker.peers {
		if peer.state == PeerStateDead {
			continue
		}
		missedHeartbeats := types.MissedTicksBetween(currentTick, peer.lastSeenAtTick)
		if missedHeartbeats >= peerTracker.suspectThreshold && peer.state == PeerStateAlive {
			updates = append(updates, peerUpdate{
				peerID:   id,
				newState: PeerStateSuspected,
				oldState: PeerStateAlive,
			})
		}
	}
	peerTracker.applyUpdates(updates)
	return updates
}

func (peerTracker *PeerTracker) confirmDeadPeersIfConsensus() []peerUpdate {
	var updates []peerUpdate
	for id := range peerTracker.peers {
		if u, ok := peerTracker.buildDeathUpdateIfConsensus(id); ok {
			updates = append(updates, u)
		}
	}
	peerTracker.applyUpdates(updates)
	return updates
}

func (peerTracker *PeerTracker) receivePeerHeartbeat(hb Heartbeat, localNodeID string, localTick uint8) []peerUpdate {
	var updates []peerUpdate

	if u, ok := peerTracker.recordHeartbeatFromSender(hb, localTick); ok {
		updates = append(updates, u)
	}

	updates = append(updates, peerTracker.updateConsensus(hb, localNodeID)...)

	peerTracker.applyUpdates(updates)

	return updates
}

func (peerTracker *PeerTracker) recordHeartbeatFromSender(hb Heartbeat, localTick uint8) (peerUpdate, bool) {
	sender, exists := peerTracker.peers[hb.PeerID]

	if !exists {
		return peerTracker.registerPeer(hb, localTick), true
	}

	if sender.state == PeerStateDead {
		return peerTracker.rejoinPeer(hb, sender, localTick), true
	}

	if types.IsSequentiallyNewer(hb.Counter, sender.lastReceivedCounter) {
		return peerTracker.refreshPeerCounters(hb, sender, localTick)
	}

	return peerUpdate{}, false
}

func (peerTracker *PeerTracker) registerPeer(hb Heartbeat, localTicker uint8) peerUpdate {
	info := &peerInfo{}
	resetPeerToAlive(info, hb, localTicker)
	peerTracker.peers[hb.PeerID] = info
	return newAliveUpdate(hb.PeerID, PeerStateDead)
}

func (peerTracker *PeerTracker) rejoinPeer(hb Heartbeat, sender *peerInfo, localTicker uint8) peerUpdate {
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

func (peerTracker *PeerTracker) refreshPeerCounters(hb Heartbeat, sender *peerInfo, localTick uint8) (peerUpdate, bool) {
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

func (peerTracker *PeerTracker) updateConsensus(hb Heartbeat, myID string) []peerUpdate {
	var updates []peerUpdate

	for _, suspectedID := range hb.SuspectedPeers {
		if suspectedID == myID {
			continue
		}
		if u, ok := peerTracker.addSuspicionAndCheckDeath(suspectedID, hb.PeerID); ok {
			updates = append(updates, u)
		}
	}

	return updates
}

func (peerTracker *PeerTracker) buildDeathUpdateIfConsensus(peerID string) (peerUpdate, bool) {
	peer, exists := peerTracker.peers[peerID]
	if !exists || peer.state != PeerStateSuspected {
		return peerUpdate{}, false
	}
	if !peerTracker.hasDeathConsensusFor(peerID) {
		return peerUpdate{}, false
	}
	return peerUpdate{
		peerID:   peerID,
		newState: PeerStateDead,
		oldState: PeerStateSuspected,
	}, true
}

func (peerTracker *PeerTracker) addSuspicionAndCheckDeath(suspectedID, reporterID string) (peerUpdate, bool) {
	peer, exists := peerTracker.peers[suspectedID]
	if !exists {
		return peerUpdate{}, false
	}

	peer.suspectedBy[reporterID] = true
	return peerTracker.buildDeathUpdateIfConsensus(suspectedID)
}

func (peerTracker *PeerTracker) applyUpdates(updates []peerUpdate) {
	for _, u := range updates {
		peer, exists := peerTracker.peers[u.peerID]
		if !exists {
			continue
		}

		peer.state = u.newState

		if u.newState == PeerStateAlive && u.oldState != PeerStateAlive {
			peer.suspectedBy = make(map[string]bool)
		}

		peerTracker.peers[u.peerID] = peer
	}
}

func (peerTracker *PeerTracker) getNonAlivePeers() []string {
	var suspected []string
	for id, peer := range peerTracker.peers {
		if peer.state == PeerStateSuspected || peer.state == PeerStateDead {
			suspected = append(suspected, id)
		}
	}
	return suspected
}