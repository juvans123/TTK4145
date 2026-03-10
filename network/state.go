package network

import (
	"heis/config"
	"heis/supervisor"
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

	var counter uint8

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
			counter++
			last.Counter = counter
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
	lastCounter := make(map[string]uint8)

	for state := range netRx {
		if state.ID == "" || state.ID == myID {
			continue
		}

		last, known := lastCounter[state.ID]
		if known && !supervisor.IsNewer(state.Counter, last){
			continue //filtrer duplikat eller forsinket
		}

		lastCounter[state.ID] = state.Counter //known settes true indirekte her
		select {
		case peerStateCh <- state:
		default:
		}
	}
}
