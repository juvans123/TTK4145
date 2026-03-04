package network

import (
	"heis/config"
)


func RunStateReceive(
	myID string,
	stateRx <-chan config.ElevatorState,
	peerStateCh chan<- config.ElevatorState,
) {
	for state := range stateRx {
        if state.ID == "" || state.ID == myID {
            continue 
        }
        peerStateCh <- state
    }
}
