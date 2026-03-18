package ordermanagement
import ("heis/config")

func initLocalElevator(ws *WorldState, myID string) {
	ws.Alive[myID] = true

	ws.States[myID] = config.ElevatorState{
		ID:          myID,
		Floor:       0,
		Behaviour:   config.BehIdle,
		Direction:   config.DirStop,
		CabRequests: make([]bool, config.N_FLOORS),
	}
}