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

type PeerUpdate struct {
	PeerID   string
	NewState PeerState
	//"OldState PeerState" kan være relevant for et ekstra lag med sikkerhet
}

type Config struct {
	MyID              string
	TickInterval      time.Duration // How often to send heartbeats and check timeouts
	SuspectThreshold  int           // Missed ticks before suspected dead
	DeadThreshold     int           // Missed ticks before dead (ingen konsensus er mulig dvs 2 heiser er død)
	ConsensusRequired int           //Antall noder som må bekrefte for å oppnå konsensus
}
