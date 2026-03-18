package fsm

import (
	"heis/elevio"
	"time"
)

func elevatorInit(e *Elevator) {
	elevio.SetDoorOpenLamp(false)
	clearAllButtonLamps()
	floor := elevio.GetFloor()
	if floor >= 0 {
		e.Floor = floor
		elevio.SetFloorIndicator(floor)
		e.Dir = elevio.MD_Stop
		e.Behavior = EB_Idle
		return
	}

	setMotor(elevio.MD_Down)

	for {
		floor := elevio.GetFloor()
		if floor >= 0 {
			stopMotor()
			e.Floor = floor
			elevio.SetFloorIndicator(floor)
			e.Dir = elevio.MD_Stop
			e.Behavior = EB_Idle
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}