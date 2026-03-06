package network

import (
	"heis/config"
	"time"
)

func RunStateBroadcast(
	myID string,
	localStateCh <-chan config.ElevatorState,
	stateTx chan<- config.ElevatorState,
) {
	period := 150 * time.Millisecond
	ticker := time.NewTicker(period)
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
			st.ID = myID
			last = st
		case <-ticker.C:
			select {
			case stateTx <- last:
			default:
			}
		}
	}
}
