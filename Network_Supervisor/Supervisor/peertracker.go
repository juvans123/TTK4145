package supervisor 


// peerInfo tracks the state of a single peer
type Peerinfo struct{
	lastCounter uint8 
	state PeerState
	suspectedBy  map[string]bool
}

// PeerTracker manages the state of all known peers
type PeerTracker struct {
    peers             map[string]*Peerinfo
    suspectThreshold  int
    deadThreshold     int
    consensusRequired int
}

func NewPeerTracker(suspectThreshold, deadThreshold, consensusRequired int) *PeerTracker {
    return &PeerTracker{
        peers:             make(map[string]*Peerinfo),
        suspectThreshold:  suspectThreshold,
        deadThreshold:     deadThreshold,
        consensusRequired: consensusRequired,
    }
}


