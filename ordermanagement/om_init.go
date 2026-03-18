package ordermanagement
import ("heis/types")

func initLocalElevator(ws *WorldState, myID string) {
	ws.Alive[myID] = true

	ws.States[myID] = types.ElevatorState{
		ID:          myID,
		Floor:       0,
		Behaviour:   types.BehIdle,
		Direction:   types.DirStop,
		CabRequests: make([]bool, types.N_FLOORS),
	}
}