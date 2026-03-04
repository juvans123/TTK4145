package supervisor

import "time"

type PeerState int

const (
	Alive PeerState = iota
	SuspectedDead
	Dead
)

func (ps PeerState) String() string {
	switch ps {
	case Alive:
		return "Alive"
	case SuspectedDead:
		return "SuspectedDead"
	case Dead:
		return "Dead"
	default:
		return "Unknown"
	}
}

type Heartbeat struct {
	PeerID         string
	Counter        uint8
	SuspectedPeers []string // Peers som denne noden betrakter som suspected dead / dead
}

type Config struct {
	MyID              string
	TickInterval      time.Duration // How often to send heartbeats and check timeouts
	SuspectThreshold  int           // Missed ticks before suspected dead
	ConsensusRequired int           //Antall noder som må bekrefte for å oppnå konsensus
}


// PeerEvent sendes til OrderManagement når en peer er bekreftet dead eller kommer tilbake
type PeerEvent struct {
	PeerID string
	Alive bool
}

func DefaultConfig(myID string) Config {
	return Config{
		MyID:              myID,
		TickInterval:      3 * time.Second,
		SuspectThreshold:  5,
		ConsensusRequired: 2,
	}
}
