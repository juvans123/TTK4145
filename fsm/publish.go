package fsm

import (
	//"fmt"
	"heis/config"
)

func tryPublishInitialState(
	stateOutCh chan<- config.ElevatorState,
	initialState config.ElevatorState,
) {
	select {
	case stateOutCh <- initialState:
	default:
	}
}

func publishStateIfChanged(
	myID string,
	elevator Elevator,
	stateOutCh chan<- config.ElevatorState,
	lastPublishedState *config.ElevatorState,
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


func equalElevatorStates(firstState, secondState config.ElevatorState) bool {
	return firstState.Floor == secondState.Floor &&
		firstState.Behaviour == secondState.Behaviour &&
		firstState.Direction == secondState.Direction &&
		firstState.Obstructed == secondState.Obstructed &&
		firstState.Immobile == secondState.Immobile &&
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