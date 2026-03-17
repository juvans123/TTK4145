package fsm

import (
	"heis/elevio"
	"heis/config"
	om "heis/ordermanagement"
	//"time"
)


func shouldStop(e *Elevator) bool {
	floor := e.Floor
	
	if (e.Dir == elevio.MD_Down && floor == 0) || (e.Dir == elevio.MD_Up && floor == config.N_FLOORS-1) {
		return true
	}

	if e.Orders.Cab[floor] {
		return true
	}
	if !om.HasOrders(&e.Orders) {
		return true
	}
	switch e.TravelDir {
	case config.TD_Up:
		if e.Orders.Hall[floor][config.BT_HallUp] {
			return true
		}
		if e.Orders.Hall[floor][config.BT_HallDown] && !om.OrdersAbove(&e.Orders, floor) {
			return true
		}
	case config.TD_Down:
		if e.Orders.Hall[floor][config.BT_HallDown] {
			return true
		}
		if e.Orders.Hall[floor][config.BT_HallUp] && !om.OrdersBelow(&e.Orders, floor) {
			return true
		}
	}
	return false
}


func chooseDirection(e *Elevator) (config.TravelDirection, Behavior, elevio.MotorDirection) {
	floor := e.Floor
	if floor < 0 {
		return e.TravelDir, EB_Idle, elevio.MD_Stop
	}

	switch e.TravelDir {
	case config.TD_Up:
		if om.OrdersAbove(&e.Orders, floor) {
			return config.TD_Up, EB_Moving, elevio.MD_Up
		}
		if om.OrdersBelow(&e.Orders, floor) {
			return config.TD_Down, EB_Moving, elevio.MD_Down
		}
	case config.TD_Down:
		if om.OrdersBelow(&e.Orders, floor) {
			return config.TD_Down, EB_Moving, elevio.MD_Down
		}
		if om.OrdersAbove(&e.Orders, floor) {
			return config.TD_Up, EB_Moving, elevio.MD_Up
		}
	}
	return e.TravelDir, EB_Idle, elevio.MD_Stop
}

func resumeTowardsLastKnownFloor(travelDir config.TravelDirection) elevio.MotorDirection {
	switch travelDir {
	case config.TD_Up:
		return elevio.MD_Down
	case config.TD_Down:
		return elevio.MD_Up
	default:
		return elevio.MD_Stop
	}
}
/* func stopMovement(
	e *Elevator,
	motorImmobileTimer *time.Timer,
	motorImmobileTimerActive *bool,
) {
	stopMotor()
	stopMotorTimer(motorImmobileTimer, motorImmobileTimerActive)
	e.Dir = elevio.MD_Stop
	e.Behavior = EB_Idle
}
 */

/* func openDoor(
	e *Elevator,
	doorTimer DoorTimer,
) {
	e.Behavior = EB_DoorOpen
	e.Dir = elevio.MD_Stop
	elevio.SetDoorOpenLamp(true)
	doorTimer.Reset(doorOpenDuration)
} */


/* func applyChosenDirection(
	e *Elevator,
	motorImmobileTimer *time.Timer,
	motorImmobileTimerActive *bool,
) {
	travelDirection, behavior, motorDirection := chooseDirection(e)
	e.TravelDir = travelDirection
	e.Behavior = behavior
	e.Dir = motorDirection

	if e.Behavior == EB_Moving {
		setMotor(e.Dir)
		startMotorTimer(motorImmobileTimer, motorImmobileTimerActive)
	}
} */