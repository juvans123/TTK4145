package network

import (
	"heis/config"
	"log"
	"time"
)

const RunStateBroadcastInterval = 300 * time.Millisecond

func RunStateBroadcast(
	myID string,
	localStateCh <-chan config.ElevatorState,
	netTx chan<- config.ElevatorState,
) {
	ticker := time.NewTicker(RunStateBroadcastInterval)
	defer ticker.Stop()

	last := config.ElevatorState{
		ID:          myID,
		Floor:       -1,
		Direction:   config.DirStop,
		Behaviour:   config.BehIdle,
		CabRequests: make([]bool, config.N_FLOORS),
	}

	for {
		select {
		case st := <-localStateCh:
			//st.ID = myID denne settes av FSM
			if st.ID != myID {
				log.Printf("networkTx localstate med uventet ID %q, men forventet %q", st.ID, myID)
				st.ID = myID
			}
			last = st
		case <-ticker.C:
			select {
			case netTx <- last:
			default:
			}
		}
	}
}

func RunStateReceive(
	myID string,
	netRx <-chan config.ElevatorState,
	peerStateCh chan<- config.ElevatorState,
) {
	for state := range netRx {
		if state.ID == "" || state.ID == myID {
			continue
		}
		select {
		case peerStateCh <- state:
		default:
		}
	}
}
