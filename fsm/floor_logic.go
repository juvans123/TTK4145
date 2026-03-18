package fsm

import (
	//"fmt"
	"heis/elevio"
	om "heis/ordermanagement"
	types "heis/types"
	"time"
)

func shouldTakeOrderInCurrentTravelDir(e *Elevator) bool {
	clear := ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
	return clear.Cab || clear.HallUp || clear.HallDown
}

func oppositeTravelDirection(dir types.TravelDirection) types.TravelDirection {
	if dir == types.TD_Up {
		return types.TD_Down
	}
	return types.TD_Up
}

func hasOrdersInTravelDirection(e *Elevator) bool {
	if e.TravelDir == types.TD_Up {
		return om.OrdersAbove(&e.Orders, e.Floor)
	}
	return om.OrdersBelow(&e.Orders, e.Floor)
}

func hasOppositeHallOrderAtFloor(e *Elevator) bool {
	if e.TravelDir == types.TD_Up {
		return e.Orders.Hall[e.Floor][types.BT_HallDown]
	}
	return e.Orders.Hall[e.Floor][types.BT_HallUp]
}

func openDoorAndStartTimer(doorTimer *time.Timer) {
	elevio.SetDoorOpenLamp(true)
	resetTimer(doorTimer, doorOpenDuration)
}

func closeDoorAndStopTimer(doorTimer *time.Timer) {
	elevio.SetDoorOpenLamp(false)
	stopTimer(doorTimer)
}

func ComputeClearEvent(orders *om.Orders, floor int, travelDir types.TravelDirection) types.ClearEvent {
	clear := types.ClearEvent{
		Floor:    floor,
		Cab:      false,
		HallUp:   false,
		HallDown: false,
	}

	if floor < 0 || floor >= len(orders.Cab) {
		return clear
	}

	hallUp := orders.Hall[floor][types.BT_HallUp]
	hallDown := orders.Hall[floor][types.BT_HallDown]

	clear.Cab = orders.Cab[floor]

	if floor == 0 {
		clear.HallUp = hallUp
		return clear
	}
	if floor == types.N_FLOORS-1 {
		clear.HallDown = hallDown
		return clear
	}

	switch travelDir {
	case types.TD_Up:
		clear.HallUp = hallUp
	case types.TD_Down:
		clear.HallDown = hallDown
	default:
		clear.HallUp = hallUp
		clear.HallDown = hallDown
	}

	return clear
}

