package network

import (
	types "heis/types"
	"time"
)

const RunStateBroadcastInterval = 100 * time.Millisecond

func BroadcastLocalState(
	myID string,
	localStateCh <-chan types.ElevatorState,
	netTx chan<- types.ElevatorState,
) {
	ticker := time.NewTicker(RunStateBroadcastInterval)
	defer ticker.Stop()

	var counter uint8
	last := initialState(myID)

	for {
		select {
		case state := <-localStateCh:
			if state.ID != myID {
				state.ID = myID
			}
			last = state

		case <-ticker.C:
			counter++
			transmitState(netTx, &last, counter)
		}
	}
}

func initialState(myID string) types.ElevatorState {
	return types.ElevatorState{
		ID:          myID,
		Floor:       -1,
		Direction:   types.DirStop,
		Behaviour:   types.BehIdle,
		CabRequests: make([]bool, types.N_FLOORS),
	}
}

func transmitState(netTx chan<- types.ElevatorState, state *types.ElevatorState, counter uint8) {
	state.Counter = counter
	netTx <- *state
}

func DeliverIncomingPeerStates(
	myID string,
	netRx <-chan types.ElevatorState,
	peerStateCh chan<- types.ElevatorState,
) {
	lastCounter := make(map[string]uint8)

	for state := range netRx {
		if state.ID == "" || state.ID == myID {
			continue
		}

		last, known := lastCounter[state.ID]
		if known && !types.IsSequentiallyNewer(state.Counter, last) {
			continue
		}

		lastCounter[state.ID] = state.Counter
		peerStateCh <- state
	}
}