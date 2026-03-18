package network

import (
	"heis/config"
	"time"
)

const RunStateBroadcastInterval = 100 * time.Millisecond

func BroadcastLocalState(
	myID string,
	localStateCh <-chan config.ElevatorState,
	netTx chan<- config.ElevatorState,
) {
	ticker := time.NewTicker(RunStateBroadcastInterval)
	defer ticker.Stop()

	var counter uint8
	last := initialState(myID)

	for {
		select {
		case st := <-localStateCh:
			last = acceptIncomingState(st, myID)

		case <-ticker.C:
			counter++
			transmitState(netTx, &last, counter)
		}
	}
}

func initialState(myID string) config.ElevatorState {
	return config.ElevatorState{
		ID:          myID,
		Floor:       -1,
		Direction:   config.DirStop,
		Behaviour:   config.BehIdle,
		CabRequests: make([]bool, config.N_FLOORS),
	}
}


func acceptIncomingState(st config.ElevatorState, myID string) config.ElevatorState {
	if st.ID != myID {
		st.ID = myID
	}
	return st
}

func transmitState(netTx chan<- config.ElevatorState, state *config.ElevatorState, counter uint8) {
	state.Counter = counter
	netTx <- *state
}

func DeliverIncomingPeerStates(
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
		if known && !config.IsSequentiallyNewer(state.Counter, last) {
			continue
		}

		lastCounter[state.ID] = state.Counter
		peerStateCh <- state
	}
}