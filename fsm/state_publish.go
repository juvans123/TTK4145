package fsm

import (
    "heis/config"
    "heis/elevio"
)

func PublicStateFromFSM(e Elevator, myID string) config.ElevatorState {
	cabCopy := make([]bool, len(e.Orders.Cab))
    copy(cabCopy, e.Orders.Cab)
    return config.ElevatorState{
        ID: myID,
        Behaviour: mapBehaviour(e.Behavior),
        Floor: e.Floor,
        Direction: mapDir(e.Dir),
        CabRequests: cabCopy,
        Obstructed: e.Obstructed,
        Immobile: e.Immobile,
    }
}

func mapBehaviour(b Behavior) config.Behaviour{
	switch b{
	case EB_Idle:
		return config.BehIdle
	case EB_Moving:
		return config.BehMoving
	case EB_DoorOpen:
		return config.BehDoorOpen
	default:
		return config.BehIdle
	}
}

func mapDir(dir elevio.MotorDirection) config.Direction{
	switch dir {
    case elevio.MD_Up:
        return config.DirUp
    case elevio.MD_Down:
        return config.DirDown
    case elevio.MD_Stop:
        return config.DirStop
    default:
        return config.DirStop
    }
}


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