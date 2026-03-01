package supervisor

import "fmt"

// peerInfo tracks the state of a single peer
type peerInfo struct {
	lastCounter uint8
	lastSeenAt  uint8 //Trenger en egen counter for å sammenlige med hb-counter altså et referansepunkt
	state       PeerState
	suspectedBy map[string]bool
}

// PeerTracker manages the state of all known peers
type peerTracker struct {
	peers             map[string]*peerInfo
	suspectThreshold  int
	deadThreshold     int
	consensusRequired int
}

type peerUpdate struct {
	peerID   string
	newState PeerState
	oldState PeerState //kan være relevant for et ekstra lag med sikkerhet
}

func NewPeerTracker(suspectThreshold, deadThreshold, consensusRequired int) *peerTracker {
	return &peerTracker{
		peers:             make(map[string]*peerInfo),
		suspectThreshold:  suspectThreshold,
		deadThreshold:     deadThreshold,
		consensusRequired: consensusRequired,
	}
}

// Funksjon som teller antall heiser i live //
func (pt *peerTracker) aliveCount() int {
	count := 0
	for _, peer := range pt.peers {
		if peer.state == Alive {
			count++
		}
	}
	return count
}

// Funksjon som sjekker for konsensus //

func (pt *peerTracker) hasConsensus(peerID string) bool {
	peer, exists := pt.peers[peerID]
	if !exists {
		return false
	}

	//Antall noder i live
	totalAlive := pt.aliveCount() + 1 // +1 inkluderer oss selv (heisen selv) 

	required := min(totalAlive, pt.consensusRequired) // Aldri krev flere enn nettverket tilbyr|

	if required < 1 {
		required = 1
	}

	suspicionCount := len(peer.suspectedBy)

	if peer.state == SuspectedDead { //Legger til vår stemme
		suspicionCount ++ 
	}

	return suspicionCount>= required

}

func (pt *peerTracker) detectHeartbeatTimeouts(myCounter uint8) []peerUpdate {
	var updates []peerUpdate
	for id, peer := range pt.peers {
		if peer.state == Dead {
			continue
		}

		missedHeartbeats := Delta(myCounter, peer.lastSeenAt) //Vår counter som referanse // -1 fordi differansen altid er >0
		oldState := peer.state
		newState := peer.state

		fmt.Printf("[PeerTracker] %s: missedHeartbeats=%d (suspect>=%d, dead>=%d) state=%s\n",
			id, missedHeartbeats, pt.suspectThreshold, pt.deadThreshold, peer.state)

		if missedHeartbeats >= pt.suspectThreshold && peer.state == Alive {
			newState = SuspectedDead
		}

		if missedHeartbeats >= pt.deadThreshold && peer.state == SuspectedDead {
			newState = Dead
		}

		if newState != oldState {
			updates = append(updates, peerUpdate{
				peerID:   id,
				newState: newState,
				oldState: oldState,
			})
		}
	}
	pt.applyUpdates(updates)

	return updates
}

func (pt *peerTracker) processHeartbeat(hb Heartbeat, myID string, myCounter uint8) []peerUpdate {
	var updates []peerUpdate

	if u, ok := pt.updateSender(hb, myCounter); ok {
		updates = append(updates, u)
	}

	updates = append(updates, pt.updateConsensus(hb, myID)...) //... pakk ut hvert element i lista og legg til individuelt

	pt.applyUpdates(updates)

	return updates
}

// updateSender håndterer tilstandsendring for avsenderen av heartbeaten
func (pt *peerTracker) updateSender(hb Heartbeat, myCounter uint8) (peerUpdate, bool) {
	sender, exists := pt.peers[hb.PeerID]

	if !exists {
		return pt.registerPeer(hb, myCounter), true
	}

	if sender.state == Dead {
		return pt.rejoinPeer(hb, sender, myCounter), true //Altså peer med state som var død begynner å sende skal rett til refresh for rejoin og ikke evalueres av Isnewer pga melding altforr lang bak
	}

	if IsNewer(hb.Counter, sender.lastCounter) { //Filtrerer bort gamle HeartBeats
		return pt.updateCounters(hb, sender, myCounter)
	}

	return peerUpdate{}, false
}

// registerPeer legger til en helt ny heis
func (pt *peerTracker) registerPeer(hb Heartbeat, myCounter uint8) peerUpdate {
	pt.peers[hb.PeerID] = &peerInfo{
		lastCounter: hb.Counter,
		lastSeenAt:  myCounter,
		state:       Alive,
		suspectedBy: make(map[string]bool), //Tom suspectedBy map
	}
	return newAliveUpdate(hb.PeerID, Dead)
}



func (pt *peerTracker) rejoinPeer (hb Heartbeat, sender *peerInfo, myCounter uint8) peerUpdate{
		oldState := sender.state
		sender.lastCounter = hb.Counter
		sender.lastSeenAt = myCounter
		sender.state = Alive 
		sender.suspectedBy = make(map[string]bool)
		return newAliveUpdate(hb.PeerID, oldState)
}

func (pt *peerTracker) updateCounters (hb Heartbeat, sender *peerInfo, myCounter uint8) (peerUpdate, bool){
	oldState := sender.state 
	sender.lastCounter = hb.Counter
	sender.lastSeenAt = myCounter

	if oldState == SuspectedDead:
	sender.state = Alive
	sender.suspectedBy = make(map[string]bool)
	return newAliveUpdate(hb.peerID oldState), true

	return peerUpdate{}, false
}


// updateConsensus behandler lista av mistenkte peers som avsenderen rapporterer
func (pt *peerTracker) updateConsensus(hb Heartbeat, myID string) []peerUpdate {
	var updates []peerUpdate

	for _, suspectedID := range hb.SuspectedPeers {
		if suspectedID == myID {
			continue
		}
		if u, ok := pt.tryConfirmDead(suspectedID, hb.PeerID); ok {
			updates = append(updates, u)
		}
	}

	return updates
}

// tryConfirmDead sjekker om konsensus er oppnådd for én mistenkt peer
// Var innkonsistent ved å sette tilstander her. Litt crazy whalla billa tilla 
func (pt *peerTracker) tryConfirmDead(suspectedID, reporterID string) (peerUpdate, bool) {
	peer, exists := pt.peers[suspectedID]
	if !exists {
		return peerUpdate{}, false
	}

	peer.suspectedBy[reporterID] = true

	if peer.state != SuspectedDead || !pt.hasConsensus(suspectedID) {
		return peerUpdate{}, false
	}

	return peerUpdate{
		peerID:   suspectedID,
		newState: Dead,
		oldState: SuspectedDead,
	}, true
}

func (pt *peerTracker) applyUpdates (updates []peerUpdate){
	for _, u := range updates {
		peer, exists := pt.peer[u.peerID]
		if !exists {
			continue
		}

		peer.state = u.newState

		if u.newState == Alive && oldState !=Alive{
			peer.suspectedBy = make(map[string]bool)
		}

		pt.peers[u.peerID] = peer
	
	}
}


func (pt *peerTracker) getSuspectedPeers() []string {
	var suspected []string
	for id, peer := range pt.peers {
		if peer.state == SuspectedDead || peer.state == Dead {
			suspected = append(suspected, id)
		}
	}
	return suspected
}

// newAliveUpdate er en hjelpefunksjon som unngår duplisert peerUpdate-konstruksjon
func newAliveUpdate(peerID string, oldState PeerState) peerUpdate {
	return peerUpdate{
		peerID:   peerID,
		newState: Alive,
		oldState: oldState,
	}
}