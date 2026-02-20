package fsm

import (
	"heis/config"
	"heis/elevio"
	om "heis/ordermanagement"
	"time"
)

 type DoorTimer interface {
	Reset(d time.Duration)
	Stop()
	Timeout() <-chan struct{}
}
const doorOpenDuration = 3 * time.Second


func Run(
	timer DoorTimer,
	floorCh <-chan int,
	omOrdersCh <-chan om.Orders,
	obstrCh <-chan bool,
	stopCh <-chan bool,
	clearCh chan<- config.ClearEvent,
) {
	e := Elevator{
		Floor:    -1,
		Dir:      config.TD_Up,
		Behavior: EB_Idle,
		Orders:   om.NewOrders(),
	}

	obstructed := false
	stopPressed := false

	for {
		select {

		// -------- Orders snapshot fra OM --------
		case newOrders := <-omOrdersCh:
			e.Orders = newOrders
			updateButtonLights(&e)

			if stopPressed {
				continue
			}

			// Hvis vi er idle og står i etasje og har ordre her: åpne dør direkte
			if e.Behavior == EB_Idle && hasOrderAtThisFloor(&e) {
				// velg “service retning” basert på dagens dir (brukes i ClearAtFloor-policy)
				if e.Floor >= 0 {
					e.Behavior = EB_DoorOpen
					openDoorAndSetLamp(timer)
					clearCh <- config.ClearEvent{Floor: e.Floor, Dir: e.Dir}
				}
				continue
			}

			// Hvis idle: start å kjøre hvis det finnes ordre et sted
			if e.Behavior == EB_Idle {
				dir, beh := chooseDirection(&e)
				e.Dir, e.Behavior = dir, beh
				if e.Behavior == EB_Moving {
					setMotor(e.Dir)
				}
			}

		// -------- Floor sensor --------
		case floor := <-floorCh:
			e.Floor = floor
			elevio.SetFloorIndicator(floor)

			if e.Behavior == EB_Moving && !stopPressed {
				if shouldStop(&e) {
					stopMotor()
					e.Behavior = EB_DoorOpen
					openDoorAndSetLamp(timer)

					// Be OM kvittere ordre i denne etasjen
					clearCh <- config.ClearEvent{Floor: e.Floor, Dir: e.Dir}
				}
			}

		// -------- Door timeout --------
		case <-timer.Timeout():
			if e.Behavior != EB_DoorOpen {
				continue
			}
			if obstructed || stopPressed {
				timer.Reset(doorOpenDuration)
				continue
			}

			closeDoorAndResetLamp(timer)
			dir, beh := chooseDirection(&e)
			e.Dir, e.Behavior = dir, beh
			if e.Behavior == EB_Moving {
				setMotor(e.Dir)
			}

		// -------- Obstruction --------
		case obs := <-obstrCh:
			obstructed = obs

		// -------- Stop button --------
		case sp := <-stopCh:
			stopPressed = sp
			elevio.SetStopLamp(sp)

			if sp {
				stopMotor()
				// Åpne dør hvis vi står i etasje
				if e.Floor >= 0 {
					e.Behavior = EB_DoorOpen
					openDoorAndSetLamp(timer)
				} else {
					e.Behavior = EB_Idle
				}
			} else {
				// Stopp sluppet: gå tilbake til normal drift
				if e.Behavior == EB_DoorOpen {
					timer.Reset(doorOpenDuration)
				} else if e.Behavior == EB_Idle {
					dir, beh := chooseDirection(&e)
					e.Dir, e.Behavior = dir, beh
					if e.Behavior == EB_Moving {
						setMotor(e.Dir)
					}
				}
			}
		}
	}
}


func setMotor(dir config.TravelDirection) {
	switch dir {
	case config.TD_Up:
		elevio.SetMotorDirection(elevio.MD_Up)
	case config.TD_Down:
		elevio.SetMotorDirection(elevio.MD_Down)
	default:
		elevio.SetMotorDirection(elevio.MD_Stop)
	}
}

func stopMotor() {
	elevio.SetMotorDirection(elevio.MD_Stop)
}

func openDoorAndSetLamp(t DoorTimer) {
	elevio.SetDoorOpenLamp(true)
	t.Reset(doorOpenDuration)
}

func closeDoorAndResetLamp(t DoorTimer) {
	elevio.SetDoorOpenLamp(false)
	t.Stop()
}


func shouldStop(e *Elevator) bool {
	floor := e.Floor
	if floor < 0 {
		return false
	}

	if e.Orders.Cab[floor]{
		return true
	}
	switch e.Dir{
	case config.TD_Up:
		if e.Orders.Hall[floor][config.BT_HallUp]{
			return true
		}
		if e.Orders.Hall[floor][config.BT_HallDown] && !om.OrdersAbove(&e.Orders, floor){
			return true
		}
	case config.TD_Down:
		if e.Orders.Hall[floor][config.BT_HallDown]{
			return true
		}
		if e.Orders.Hall[floor][config.BT_HallUp] && !om.OrdersBelow(&e.Orders, floor){
			return true
		}
	}
	return false
}

func chooseDirection(e *Elevator) (config.TravelDirection, Behavior){
	floor := e.Floor
	if floor < 0{
		return e.Dir, EB_Idle
	}

	switch e.Dir {
	case config.TD_Up:
		if om.OrdersAbove(&e.Orders, floor) {
			return config.TD_Up, EB_Moving
		}
		if om.OrdersBelow(&e.Orders, floor) {
			return config.TD_Down, EB_Moving
		}
	case config.TD_Down:
		if om.OrdersBelow(&e.Orders, floor) {
			return config.TD_Down, EB_Moving
		}
		if om.OrdersAbove(&e.Orders, floor) {
			return config.TD_Up, EB_Moving
		}
	}
	return e.Dir, EB_Idle
}

func hasOrderAtThisFloor(e *Elevator) bool {
	if e.Floor < 0 {
		return false
	}
	return om.HasOrderAtFloor(&e.Orders, e.Floor)
}

func updateButtonLights(e *Elevator) {
	for floor := 0; floor < len(e.Orders.Cab); floor++ {
		elevio.SetButtonLamp(config.BT_Cab, floor, e.Orders.Cab[floor])
		elevio.SetButtonLamp(config.BT_HallUp, floor, e.Orders.Hall[floor][config.BT_HallUp])
		elevio.SetButtonLamp(config.BT_HallDown, floor, e.Orders.Hall[floor][config.BT_HallDown])
	}
}
