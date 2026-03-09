package fsm

import (
	"fmt"
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
	setButtonLight <- chan om.WorldState,
) {
	e := Elevator{
		Floor:    -1,
		Dir:      elevio.MD_Down,
		TravelDir: config.TD_Down,
		Behavior: EB_Moving,
		Orders:   om.NewOrders(4),
	}

	obstructed := false
	stopPressed := false
	elevatorInit(&e)
	publishState(myID, e, stateOutCh)

	period := 5000 * time.Millisecond
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {

		// -------- Orders snapshot fra OM --------
		case newOrders := <-omOrdersCh:
			//prevAtFloor := (e.Floor >= 0) && om.HasOrderAtFloor(&e.Orders, e.Floor)
			//prevAtFloor unngår spam når om SENDER OPPDATERINGER OFTERE ENN vi trykker, fiks denne når det blir relevant
			e.Orders = newOrders
			//updateButtonLights(&e)
			if stopPressed {
				publishState(myID, e, stateOutCh)
				continue
			}

			// Hvis døra er åpen og vi fikk ny ordre i samme etasje: hold døra åpen og clear
			if e.Behavior == EB_DoorOpen && e.Floor >= 0 && om.HasOrderAtFloor(&e.Orders, e.Floor) { //sett in prevAtFloor
				timer.Reset(doorOpenDuration)
				fmt.Printf("reset dør")
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				continue
			}

			// Hvis idle i etasje og har ordre her: åpne
			if e.Behavior == EB_Idle && e.Floor >= 0 && om.HasOrderAtFloor(&e.Orders, e.Floor) {
				e.Behavior = EB_DoorOpen
				openDoorAndSetLamp(timer)
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				continue
			}

			if e.Behavior == EB_Idle {
				travelDir, behavior, dir := chooseDirection(&e)
				e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
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
					e.Dir = elevio.MD_Stop
					e.Behavior = EB_Idle
					fmt.Printf("[FSM %s] Got orders, behavior=%v, floor=%d, traveldir=%v, direction=%v\n", myID, e.Behavior, e.Floor, e.TravelDir, e.Dir)
					if om.HasOrderAtFloor(&e.Orders, e.Floor) {
						e.Behavior = EB_DoorOpen
						openDoorAndSetLamp(timer)
						ce := ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
						clearCh <- ce
					} /* else {
						// Boundary stop without an order at this floor: choose a new valid direction.
						dir, beh := chooseDirection(&e)
						e.Dir, e.Behavior = dir, beh
						if e.Behavior == EB_Moving {
							setMotor(e.Dir)
						}
					} */
				}
			}
			publishState(myID, e, stateOutCh)

		// -------- Button light update --------
		case worldState := <-setButtonLight:
			updateButtonLights(&worldState, myID)
			

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

			travelDir, behavior, dir := chooseDirection(&e)
			e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
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
				e.Dir = elevio.MD_Stop
				e.Behavior = EB_Idle
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

					travelDir, behavior, dir := chooseDirection(&e)
					e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
					if e.Behavior == EB_Moving {
						setMotor(e.Dir)
					}
				}
			}
			publishState(myID, e, stateOutCh)
		case <-ticker.C:
			//fmt.Println("Im Alive: %s", myID)
		}
	}
}

func elevatorInit(e *Elevator) {
	//updateButtonLights(e)
	setMotor(elevio.MD_Down)
	for {
		if elevio.GetFloor() >= 0 {
			stopMotor()
			e.Dir = elevio.MD_Stop
			e.Behavior = EB_Idle
			break
		}
	}
}

func setMotor(dir elevio.MotorDirection) {
	switch dir {
	case elevio.MD_Up:
		elevio.SetMotorDirection(elevio.MD_Up)

	case elevio.MD_Down:
		elevio.SetMotorDirection(elevio.MD_Down)
	case elevio.MD_Stop:
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

	// Never continue past the end floors.
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

func updateButtonLights(ws *om.WorldState, myID string) {
	for floor := 0; floor < len(ws.ConfirmedCabOrders); floor++ {
		elevio.SetButtonLamp(config.BT_Cab, floor, ws.ConfirmedCabOrders[myID][floor])
		elevio.SetButtonLamp(config.BT_HallUp, floor, ws.ConfirmedHallOrders[floor][config.BT_HallUp])
		elevio.SetButtonLamp(config.BT_HallDown, floor, ws.ConfirmedHallOrders[floor][config.BT_HallDown])
	}
}

func ComputeClearEvent(orders *om.Orders, floor int, dir config.TravelDirection) config.ClearEvent {
	ce := config.ClearEvent{
		Floor:         floor,
		ClearCab:      true, //alltid clear cab på etasje
		ClearHallUp:   false,
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
	fmt.Println(st)

	// Ikke blokker FSM hvis mottaker henger
	select {
	case stateOutCh <- st:
	default:
	}
}
