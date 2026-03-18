package fsm

import (
	"heis/elevio"
	types "heis/types"
	om "heis/ordermanagement"
	//"time"
)


func shouldStop(e *Elevator) bool {
	floor := e.Floor
	
	if floor == 0 && (e.Dir == elevio.MD_Down || e.Dir == elevio.MD_Stop) {
		return true
	}
	if floor == types.N_FLOORS-1 && (e.Dir == elevio.MD_Up || e.Dir == elevio.MD_Stop) {
		return true
	}

	if e.Orders.Cab[floor] {
		return true
	}
	if !om.HasOrders(&e.Orders) {
		return true
	}
	switch e.TravelDir {
	case types.TD_Up:
		if e.Orders.Hall[floor][types.BT_HallUp] {
			return true
		}
		if e.Orders.Hall[floor][types.BT_HallDown] && !om.OrdersAbove(&e.Orders, floor) {
			return true
		}
	case types.TD_Down:
		if e.Orders.Hall[floor][types.BT_HallDown] {
			return true
		}
		if e.Orders.Hall[floor][types.BT_HallUp] && !om.OrdersBelow(&e.Orders, floor) {
			return true
		}
	}
	return false
}


func chooseDirection(e *Elevator) (types.TravelDirection, Behavior, elevio.MotorDirection) {
	floor := e.Floor
	switch e.TravelDir {
	case types.TD_Up:
		if om.OrdersAbove(&e.Orders, floor) {
			return types.TD_Up, EB_Moving, elevio.MD_Up
		}
		if om.OrdersBelow(&e.Orders, floor) {
			return types.TD_Down, EB_Moving, elevio.MD_Down
		}
	case types.TD_Down:
		if om.OrdersBelow(&e.Orders, floor) {
			return types.TD_Down, EB_Moving, elevio.MD_Down
		}
		if om.OrdersAbove(&e.Orders, floor) {
			return types.TD_Up, EB_Moving, elevio.MD_Up
		}
	}
	return e.TravelDir, EB_Idle, elevio.MD_Stop
}

func resumeTowardsLastKnownFloor(travelDir types.TravelDirection) elevio.MotorDirection {
	switch travelDir {
	case types.TD_Up:
		return elevio.MD_Down
	case types.TD_Down:
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