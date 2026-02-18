package supervisor

type Supervisor struct {
	config    Config
	myCounter uint8

	hbTx         chan<- Heartbeat
	hbRx         <-chan Heartbeat
	peerUpdateTx chan<- PeerUpdate

	stateCh chan chan map[string]PeerState //Finn ut
}

func New(
	cfg Config,
	hbTx chan<- Heartbeat,
	hbRx <-chan Heartbeat,
	peerUpdateTx chan<- PeerUpdate,
) *Supervisor {
	return &Supervisor{
		config:       cfg,
		myCounter:    0,
		hbTx:         hbTx,
		hbRx:         hbRx,
		peerUpdateTx: peerUpdateTx,
		stateCh:      make(chan chan map[string]PeerState),
	}
}


