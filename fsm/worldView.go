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