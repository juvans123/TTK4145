package fsm

import "heis/config"

func PublicStateFromFSM(e Elevator, myID string) config.ElevatorState {
	cabCopy := make([]bool, len(e.Orders.Cab))
    copy(cabCopy, e.Orders.Cab)
    return config.ElevatorState{
        ID: myID,
        Behaviour: mapBehaviour(e.Behavior),
        Floor: e.Floor,
        Direction: mapDir(e.Dir),
        CabRequests: cabCopy, 
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

func mapDir(d config.TravelDirection) config.Direction{
	switch d {
    case config.TD_Up:
        return config.DirUp
    case config.TD_Down:
        return config.DirDown
    case config.TD_Stop:
        return config.DirStop
    default:
        return config.DirStop
    }
}