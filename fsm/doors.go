package fsm

import (
	"heis/config"
	"heis/elevio"
	om "heis/ordermanagement"
	"time"
)

func openDoorAndStartTimer(doorTimer *time.Timer) {
	elevio.SetDoorOpenLamp(true)
	resetTimer(doorTimer, doorOpenDuration)
}

func closeDoorAndStopTimer(doorTimer *time.Timer) {
	elevio.SetDoorOpenLamp(false)
	stopTimerChannel(doorTimer)
}

func ComputeClearEvent(orders *om.Orders, floor int, dir config.TravelDirection) config.ClearEvent {
	ce := config.ClearEvent{
		Floor:         floor,
		ClearCab:      false,
		ClearHallUp:   false,
		ClearHallDown: false,
	}

	if floor < 0 || floor >= len(orders.Cab) {
		return ce
	}

	hallUp := orders.Hall[floor][config.BT_HallUp]
	hallDown := orders.Hall[floor][config.BT_HallDown]

	ce.ClearCab = orders.Cab[floor]

	if floor == 0 {
		ce.ClearHallUp = hallUp
		return ce
	}
	if floor == config.N_FLOORS-1 {
		ce.ClearHallDown = hallDown
		return ce
	}

	switch dir {
	case config.TD_Up:
		if hallUp {
			ce.ClearHallUp = true
		}

	case config.TD_Down:
		if hallDown {
			ce.ClearHallDown = true
		}

	default:
		ce.ClearHallUp = hallUp
		ce.ClearHallDown = hallDown
	}

	return ce
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