package fsm

import (
	//"fmt"
	"heis/config"
	om "heis/ordermanagement"
	"heis/elevio"
	"time"
)

func shouldTakeOrderInCurrentTravelDir(e *Elevator) bool {
	clear := ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
	return clear.Cab || clear.HallUp || clear.HallDown
}

func oppositeTravelDirection(dir config.TravelDirection) config.TravelDirection {
	if dir == config.TD_Up {
		return config.TD_Down
	}
	return config.TD_Up
}

func hasOrdersInTravelDirection(e *Elevator) bool {
	if e.TravelDir == config.TD_Up {
		return om.OrdersAbove(&e.Orders, e.Floor)
	}
	return om.OrdersBelow(&e.Orders, e.Floor)
}

func hasOppositeHallOrderAtFloor(e *Elevator) bool {
	if e.TravelDir == config.TD_Up {
		return e.Orders.Hall[e.Floor][config.BT_HallDown]
	}
	return e.Orders.Hall[e.Floor][config.BT_HallUp]
}

func openDoorAndStartTimer(doorTimer *time.Timer) {
	elevio.SetDoorOpenLamp(true)
	resetTimer(doorTimer, doorOpenDuration)
}

func closeDoorAndStopTimer(doorTimer *time.Timer) {
	elevio.SetDoorOpenLamp(false)
	stopTimerChannel(doorTimer)
}

func ComputeClearEvent(orders *om.Orders, floor int, travelDir config.TravelDirection) config.ClearEvent {
	clear := config.ClearEvent{
		Floor:    floor,
		Cab:      false,
		HallUp:   false,
		HallDown: false,
	}

	if floor < 0 || floor >= len(orders.Cab) {
		return clear
	}

	hallUp := orders.Hall[floor][config.BT_HallUp]
	hallDown := orders.Hall[floor][config.BT_HallDown]

	clear.Cab = orders.Cab[floor]

	if floor == 0 {
		clear.HallUp = hallUp
		return clear
	}
	if floor == config.N_FLOORS-1 {
		clear.HallDown = hallDown
		return clear
	}
	
	switch travelDir {
	case config.TD_Up:
		clear.HallUp = hallUp
	case config.TD_Down:
		clear.HallDown = hallDown
	default:
		clear.HallUp = hallUp
		clear.HallDown = hallDown
	}

	return clear
}


/* func ComputeClearEvent(orders *om.Orders, floor int, travelDirection config.TravelDirection) config.ClearEvent {
	clearEvent := config.ClearEvent{Floor: floor}

	if orders.Cab[floor] {
		clearEvent.Buttons = append(clearEvent.Buttons, elevio.BT_Cab)
	}
	if shouldTakeOrderInCurrentTravelDir(orders, floor, config.BT_HallUp, travelDirection) {
		clearEvent.Buttons = append(clearEvent.Buttons, elevio.BT_HallUp)
	}
	if shouldTakeOrderInCurrentTravelDir(orders, floor, config.BT_HallDown, travelDirection) {
		clearEvent.Buttons = append(clearEvent.Buttons, elevio.BT_HallDown)
	}

	return clearEvent
} */