package fsm

import (
	"heis/config"
	"heis/elevio"
	om "heis/ordermanagement"
)

func openDoorAndStartTimer(doorTimer DoorTimer) {
	elevio.SetDoorOpenLamp(true)
	doorTimer.Reset(doorOpenDuration)
}

func closeDoorAndStopTimer(doorTimer DoorTimer) {
	elevio.SetDoorOpenLamp(false)
	doorTimer.Stop()
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