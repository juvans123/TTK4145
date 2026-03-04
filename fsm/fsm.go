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
	myID string,
	timer DoorTimer,
	floorCh <-chan int,
	omOrdersCh <-chan om.Orders,
	obstrCh <-chan bool,
	stopCh <-chan bool,
	clearCh chan<- config.ClearEvent,
	stateOutCh chan<- config.ElevatorState,
) {
	e := Elevator{
		Floor:    -1,
		Dir:      config.TD_Down,
		Behavior: EB_Moving,
		Orders:   om.NewOrders(4),
	}

	obstructed := false
	stopPressed := false
	elevatorInit(&e)
	publishState(myID, e, stateOutCh)


	for {
		select {

		// -------- Orders snapshot fra OM --------
		case newOrders := <-omOrdersCh:
			prevAtFloor := (e.Floor >= 0) && om.HasOrderAtFloor(&e.Orders, e.Floor)
			e.Orders = newOrders
			updateButtonLights(&e)
			if stopPressed {
				publishState(myID, e, stateOutCh)
				continue
			}

			// Hvis døra er åpen og vi fikk ny ordre i samme etasje: hold døra åpen og clear
			if e.Behavior == EB_DoorOpen && e.Floor >= 0 && om.HasOrderAtFloor(&e.Orders, e.Floor)  && !prevAtFloor {
				timer.Reset(doorOpenDuration)
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.Dir)
				continue
			}
		
			// Hvis idle i etasje og har ordre her: åpne
			if e.Behavior == EB_Idle && e.Floor >= 0 && om.HasOrderAtFloor(&e.Orders, e.Floor) {
				e.Behavior = EB_DoorOpen
				openDoorAndSetLamp(timer)
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.Dir)
				continue
			}

			if e.Behavior == EB_Idle {
				dir, beh := chooseDirection(&e)
				e.Dir, e.Behavior = dir, beh
				if e.Behavior == EB_Moving {
					setMotor(e.Dir)
				}
			}
			publishState(myID, e, stateOutCh)

		// -------- Floor sensor --------
		case floor := <-floorCh:
			e.Floor = floor
			elevio.SetFloorIndicator(floor)

			if e.Behavior == EB_Moving && !stopPressed {
				if shouldStop(&e) {
					stopMotor()
					e.Behavior = EB_DoorOpen
					openDoorAndSetLamp(timer)
					ce := ComputeClearEvent(&e.Orders, e.Floor, e.Dir)
					clearCh <- ce
				}
			}
			publishState(myID, e, stateOutCh)

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
			publishState(myID, e, stateOutCh)

		// -------- Obstruction --------
		case obs := <-obstrCh:
			obstructed = obs

		// -------- Stop button --------
		case sp := <-stopCh:
			stopPressed = sp
			elevio.SetStopLamp(sp)

			if sp {
				stopMotor()
				floor := elevio.GetFloor()

				if floor >= 0 {
					e.Behavior = EB_DoorOpen
					openDoorAndSetLamp(timer)
					//clearer ikke her
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
			publishState(myID, e, stateOutCh)
		}
	}
}

func elevatorInit(e *Elevator){
	updateButtonLights(e)
	setMotor(config.TD_Down)
	for {
		if elevio.GetFloor() >= 0{
			stopMotor()
			e.Behavior = EB_Idle
			break
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


func updateButtonLights(e *Elevator) {
	for floor := 0; floor < len(e.Orders.Cab); floor++ {
		elevio.SetButtonLamp(config.BT_Cab, floor, e.Orders.Cab[floor])
		elevio.SetButtonLamp(config.BT_HallUp, floor, e.Orders.Hall[floor][config.BT_HallUp])
		elevio.SetButtonLamp(config.BT_HallDown, floor, e.Orders.Hall[floor][config.BT_HallDown])
	}
}

func ComputeClearEvent(orders *om.Orders, floor int, dir config.TravelDirection) config.ClearEvent {
	ce := config.ClearEvent{
		Floor: floor,
		ClearCab: true, //alltid clear cab på etasje
		ClearHallUp: false,
		ClearHallDown: false,
	}
	hallUp := orders.Hall[floor][config.BT_HallUp]
	hallDown := orders.Hall[floor][config.BT_HallDown]

	switch dir {
	case config.TD_Up:
		if hallUp {
			ce.ClearHallUp = true
		}
		if hallDown && !om.OrdersAbove(orders, floor) {
			ce.ClearHallDown = true
		}

	case config.TD_Down:
		if hallDown {
			ce.ClearHallDown = true
		}
		if hallUp && !om.OrdersBelow(orders, floor) {
			ce.ClearHallUp = true
		}

	/* default:
		ce.ClearHallUp = hallUp
		ce.ClearHallDown = hallDown */
	}

	return ce
}

func publishState(myID string, e Elevator, stateOutCh chan<- config.ElevatorState) {
    st := PublicStateFromFSM(e, myID)

    // Ikke blokker FSM hvis mottaker henger
    select {
    case stateOutCh <- st:
    default:
    }
}