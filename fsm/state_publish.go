package fsm

import (
	"heis/elevio"
	types "heis/types"
)

func PublicStateFromFSM(e Elevator, myID string) types.ElevatorState {
	cabCopy := make([]bool, len(e.Orders.Cab))
	copy(cabCopy, e.Orders.Cab)
	return types.ElevatorState{
		ID:          myID,
		Behaviour:   mapBehaviour(e.Behavior),
		Floor:       e.Floor,
		Direction:   mapDir(e.Dir),
		CabRequests: cabCopy,
		Obstructed:  e.IsObstructed,
		IsImmobile:  e.IsImmobile,
	}
}

func mapBehaviour(b Behavior) types.Behaviour {
	switch b {
	case EB_Idle:
		return types.BehIdle
	case EB_Moving:
		return types.BehMoving
	case EB_DoorOpen:
		return types.BehDoorOpen
	default:
		return types.BehIdle
	}
}

func mapDir(dir elevio.MotorDirection) types.Direction {
	switch dir {
	case elevio.MD_Up:
		return types.DirUp
	case elevio.MD_Down:
		return types.DirDown
	case elevio.MD_Stop:
		return types.DirStop
	default:
		return types.DirStop
	}
}

func tryPublishInitialState(
	stateOutCh chan<- types.ElevatorState,
	initialState types.ElevatorState,
) {
	select {
	case stateOutCh <- initialState:
	default:
	}
}

func publishStateIfChanged(
	myID string,
	elevator Elevator,
	stateOutCh chan<- types.ElevatorState,
	lastPublishedState *types.ElevatorState,
) {
	currentState := PublicStateFromFSM(elevator, myID)
	if equalElevatorStates(currentState, *lastPublishedState) {
		return
	}

	select {
	case stateOutCh <- currentState:
		/* fmt.Printf(
			"[FSM %s] PUBLISH state floor=%d beh=%v dir=%v cab=%v\n",
			myID,
			currentState.Floor,
			currentState.Behaviour,
			currentState.Direction,
			currentState.CabRequests,
		) */
		*lastPublishedState = currentState
	default:
	}
}

func equalElevatorStates(firstState, secondState types.ElevatorState) bool {
	return firstState.Floor == secondState.Floor &&
		firstState.Behaviour == secondState.Behaviour &&
		firstState.Direction == secondState.Direction &&
		firstState.Obstructed == secondState.Obstructed &&
		firstState.IsImmobile == secondState.IsImmobile &&
		equalCabRequests(firstState.CabRequests, secondState.CabRequests)
}

func equalCabRequests(first, second []bool) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}
