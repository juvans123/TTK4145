package supervisor

import (
	"testing"
)

// ─── Hjelpefunksjoner ───────────────────────────────────────────────────────

func newTracker() *peerTracker {
	return NewPeerTracker(5, 2)
}

// Registrerer en peer manuelt med gitt state og lastSeenAt
func seedPeer(pt *peerTracker, id string, state PeerState, lastSeenAt uint8) {
	pt.peers[id] = &peerInfo{
		lastCounter: 1,
		lastSeenAt:  lastSeenAt,
		state:       state,
		suspectedBy: make(map[string]bool),
	}
}

func assertState(t *testing.T, pt *peerTracker, peerID string, expected PeerState) {
	t.Helper()
	peer, exists := pt.peers[peerID]
	if !exists {
		t.Errorf("peer %s finnes ikke", peerID)
		return
	}
	if peer.state != expected {
		t.Errorf("peer %s: forventet %s, fikk %s", peerID, expected, peer.state)
	}
}

func assertUpdatesContain(t *testing.T, updates []peerUpdate, peerID string, newState PeerState) {
	t.Helper()
	for _, u := range updates {
		if u.peerID == peerID && u.newState == newState {
			return
		}
	}
	t.Errorf("forventet update {%s -> %s}, fant ikke i: %v", peerID, newState, updates)
}

func assertNoUpdate(t *testing.T, updates []peerUpdate) {
	t.Helper()
	if len(updates) != 0 {
		t.Errorf("forventet ingen updates, fikk: %v", updates)
	}
}

// ─── hasConsensus ───────────────────────────────────────────────────────────

func TestHasConsensus_IngenStemmer(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Alive, 0)
	// Alive: vår stemme teller ikke, ingen andre har rapportert
	if pt.hasConsensus("B") {
		t.Error("skal ikke ha konsensus når B er Alive og ingen har rapportert")
	}
}

func TestHasConsensus_KunEgenStemme_SuspectedDead(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", SuspectedDead, 0)
	// Vi er den eneste noden (aliveCount=0 -> effectiveRequired=1), vår stemme holder
	if !pt.hasConsensus("B") {
		t.Error("skal ha konsensus når vi er eneste node og B er SuspectedDead")
	}
}

func TestHasConsensus_TreNoder_KrevToStemmer(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", SuspectedDead, 0)
	seedPeer(pt, "C", Alive, 0) // C er i live -> aliveCount=1 -> effectiveRequired=2

	// Kun vår stemme (suspicionCount=1) - ikke nok
	if pt.hasConsensus("B") {
		t.Error("skal ikke ha konsensus med kun 1 stemme når 2 kreves")
	}

	// C rapporterer B som suspected
	pt.peers["B"].suspectedBy["C"] = true
	if !pt.hasConsensus("B") {
		t.Error("skal ha konsensus når C også mistenker B")
	}
}

func TestHasConsensus_PeerFinnesIkke(t *testing.T) {
	pt := newTracker()
	if pt.hasConsensus("ukjent") {
		t.Error("skal returnere false for ukjent peer")
	}
}

// ─── tryConfirmDead ─────────────────────────────────────────────────────────

func TestTryConfirmDead_IkkeNokStemmer(t *testing.T) {
	pt := NewPeerTracker(5,3)
	seedPeer(pt, "B", SuspectedDead, 0)
	seedPeer(pt, "C", Alive, 0) // trenger 2 stemmer
	seedPeer(pt, "D", Alive, 0)


	// Bare C rapporterer, men vi har ikke selv mistenkt ennå - nei, B er SuspectedDead så vi teller
	// suspicionCount = 1 (vår) + 1 (C) = 2, aliveCount=1, effectiveRequired=2 -> konsensus!
	// Denne testen: bare én ekstern reporter, ingen andre alive
	pt2 := newTracker()
	seedPeer(pt2, "B", SuspectedDead, 0)
	seedPeer(pt2, "C", Alive, 0)
	seedPeer(pt2, "D", Alive, 0) // nå er effectiveRequired=2, men vi trenger C+D

	u, ok := pt2.tryConfirmDead("B", "C") // kun C stemmer
	if ok {
		t.Errorf("skal ikke bekrefte dead med kun 1 ekstern stemme, fikk: %v", u)
	}
	assertState(t, pt2, "B", SuspectedDead)
}

func TestTryConfirmDead_NokStemmer(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", SuspectedDead, 0)
	seedPeer(pt, "C", Alive, 0)

	// C rapporterer B: suspicionCount = vår(1) + C(1) = 2, effectiveRequired=2
	u, ok := pt.tryConfirmDead("B", "C")
	if !ok {
		t.Error("skal bekrefte dead når konsensus er nådd")
	}
	if u.newState != Dead {
		t.Errorf("forventet Dead, fikk %s", u.newState)
	}
}

func TestTryConfirmDead_PeerErAlive_IkkeKonfirmer(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Alive, 0) // B er ikke suspected

	_, ok := pt.tryConfirmDead("B", "C")
	if ok {
		t.Error("skal ikke bekrefte dead for en Alive peer")
	}
}

func TestTryConfirmDead_DuplikatStemme_TellesEnGang(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", SuspectedDead, 0)
	seedPeer(pt, "C", Alive, 0)
	seedPeer(pt, "D", Alive, 0) // effectiveRequired=2, trenger vår + C eller D

	// C stemmer to ganger (f.eks. forsinket duplikat)
	pt.tryConfirmDead("B", "C")
	pt.tryConfirmDead("B", "C") // duplikat

	// Fortsatt kun C sin stemme + vår = 2, men D er alive -> effectiveRequired=min(3,2)=2
	// Skal nå ha konsensus siden vi har vår + C = 2
	if !pt.hasConsensus("B") {
		t.Error("skal ha konsensus: vår stemme + C = 2")
	}
}

func TestTryConfirmDead_PeerFinnesIkke(t *testing.T) {
	pt := newTracker()
	_, ok := pt.tryConfirmDead("ukjent", "C")
	if ok {
		t.Error("skal returnere false for ukjent peer")
	}
}

// ─── checkConsensusForSuspected ─────────────────────────────────────────────

func TestCheckConsensus_EnesteNode_KonfirmererDead(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", SuspectedDead, 0)
	// Ingen andre alive -> vi er eneste node -> effectiveRequired=1

	updates := pt.checkConsensusForSuspected()
	assertUpdatesContain(t, updates, "B", Dead)
	assertState(t, pt, "B", Dead)
}

func TestCheckConsensus_AliveNodeBliIkkeKonfirmert(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Alive, 0)

	updates := pt.checkConsensusForSuspected()
	assertNoUpdate(t, updates)
}

func TestCheckConsensus_FlereSuspected_KonfirmererBegge(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", SuspectedDead, 0)
	seedPeer(pt, "C", SuspectedDead, 0)
	// Ingen alive -> effectiveRequired=1 for begge

	updates := pt.checkConsensusForSuspected()
	assertUpdatesContain(t, updates, "B", Dead)
	assertUpdatesContain(t, updates, "C", Dead)
}

// ─── detectHeartbeatTimeouts ────────────────────────────────────────────────

func TestDetectTimeouts_AliveBlirSuspected(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Alive, 0) // lastSeenAt=0

	updates := pt.detectHeartbeatTimeouts(5) // missedHeartbeats=5 >= threshold=5
	assertUpdatesContain(t, updates, "B", SuspectedDead)
}

func TestDetectTimeouts_IkkeNokMissed(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Alive, 0)

	updates := pt.detectHeartbeatTimeouts(3) // missedHeartbeats=3 < threshold=5
	assertNoUpdate(t, updates)
}

func TestDetectTimeouts_DeadBliIgnorert(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Dead, 0)

	updates := pt.detectHeartbeatTimeouts(100)
	assertNoUpdate(t, updates)
}

func TestDetectTimeouts_SuspectedOgEnesteNode_BlirDead(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Alive, 0)

	// Tick 5: B blir SuspectedDead, ingen andre alive -> konsensus -> Dead
	updates := pt.detectHeartbeatTimeouts(5)
	assertUpdatesContain(t, updates, "B", SuspectedDead)
	assertUpdatesContain(t, updates, "B", Dead)
	assertState(t, pt, "B", Dead)
}

// ─── Stresstest: motstridende/forsinkede meldinger ──────────────────────────

func TestStress_ForsinketHeartbeat_GjenopplivingAvDead(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Dead, 0)

	// Forsinket heartbeat fra B (var dead, sender nå igjen)
	hb := Heartbeat{PeerID: "B", Counter: 50, SuspectedPeers: nil}
	updates := pt.processHeartbeat(hb, "A", 10)
	assertUpdatesContain(t, updates, "B", Alive)
	assertState(t, pt, "B", Alive)
}

func TestStress_GammelHeartbeat_BlirIgnorert(t *testing.T) {
	pt := newTracker()
	pt.peers["B"] = &peerInfo{
		lastCounter: 100,
		lastSeenAt:  10,
		state:       Alive,
		suspectedBy: make(map[string]bool),
	}

	// Counter 50 er eldre enn 100 -> skal ignoreres
	hb := Heartbeat{PeerID: "B", Counter: 50, SuspectedPeers: nil}
	updates := pt.processHeartbeat(hb, "A", 15)
	assertNoUpdate(t, updates)
	assertState(t, pt, "B", Alive)
}

func TestStress_MoistridendeSuspicionOgHeartbeat(t *testing.T) {
	// C rapporterer B som suspected, men B sender fortsatt heartbeat
	pt := newTracker()
	seedPeer(pt, "B", SuspectedDead, 5)
	seedPeer(pt, "C", Alive, 5)

	// C rapporterer B som dead i heartbeat
	hbFromC := Heartbeat{PeerID: "C", Counter: 10, SuspectedPeers: []string{"B"}}
	pt.processHeartbeat(hbFromC, "A", 6)

	// B sender en ny heartbeat (counter=10 > lastCounter=1)
	hbFromB := Heartbeat{PeerID: "B", Counter: 10, SuspectedPeers: nil}
	updates := pt.processHeartbeat(hbFromB, "A", 6)

	// B er i live igjen pga ny heartbeat
	assertUpdatesContain(t, updates, "B", Alive)
	assertState(t, pt, "B", Alive)
}

func TestStress_TreHeiser_ToDoer_SisteKonfirmererBegge(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Alive, 0)
	seedPeer(pt, "C", Alive, 0)

	// Tick 5: B og C blir suspected
	pt.detectHeartbeatTimeouts(5)
	assertState(t, pt, "B", SuspectedDead)
	assertState(t, pt, "C", SuspectedDead)

	// Ingen andre alive -> vi er eneste -> begge Dead via checkConsensusForSuspected
	assertState(t, pt, "B", Dead)
	assertState(t, pt, "C", Dead)
}

func TestStress_GjentagendeStemmerEndrerIkkeState(t *testing.T) {
	pt := newTracker()
	seedPeer(pt, "B", Dead, 0)

	// Forsinkede rapporter om B etter at B allerede er Dead
	pt.tryConfirmDead("B", "C")
	pt.tryConfirmDead("B", "C")
	pt.tryConfirmDead("B", "D")

	assertState(t, pt, "B", Dead) // fortsatt Dead, ingen endring
}

func TestStress_CounterWrapAround(t *testing.T) {
	pt := newTracker()
	pt.peers["B"] = &peerInfo{
		lastCounter: 250,
		lastSeenAt:  250,
		state:       Alive,
		suspectedBy: make(map[string]bool),
	}

	// Counter wrapper fra 250 -> 10 (IsNewer skal håndtere dette)
	hb := Heartbeat{PeerID: "B", Counter: 10, SuspectedPeers: nil}
	updates := pt.processHeartbeat(hb, "A", 251)
	assertUpdatesContain(t, updates, "B", Alive) // Skal regnes som nyere pga wrap
}
