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
	setButtonLight <-chan config.LightState,
) {
	e := Elevator{
		Floor:     -1,
		Dir:       elevio.MD_Down,
		TravelDir: config.TD_Down,
		Behavior:  EB_Moving,
		Orders:    om.NewOrders(4),
	}

	obstructed := false
	stopPressed := false
	elevatorInit(&e)

	lastPublished := PublicStateFromFSM(e, myID)
	select {
	case stateOutCh <- lastPublished:
	default:
	}

	period := 5000 * time.Millisecond
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	publishIfChanged := func() {
		st := PublicStateFromFSM(e, myID)
		if st.Floor != lastPublished.Floor ||
			st.Behaviour != lastPublished.Behaviour ||
			st.Direction != lastPublished.Direction {
			select {
			case stateOutCh <- st:
				lastPublished = st
			default:
			}
		}
	}

	for {
		select {

		// -------- Orders snapshot fra OM --------
		case newOrders := <-omOrdersCh:
			//fmt.Printf("orders:, %v\n", newOrders)
			//prevAtFloor := (e.Floor >= 0) && om.HasOrderAtFloor(&e.Orders, e.Floor)
			//prevAtFloor unngår spam når om SENDER OPPDATERINGER OFTERE ENN vi trykker, fiks denne når det blir relevant
			e.Orders = newOrders
			//updateButtonLights(&e)
			if stopPressed {
				publishIfChanged()
				continue
			}

			// Hvis døra er åpen og vi fikk ny ordre i samme etasje: hold døra åpen og clear
			if e.Behavior == EB_DoorOpen && e.Floor >= 0 && om.HasOrderAtFloor(&e.Orders, e.Floor) { //sett in prevAtFloor
				timer.Reset(doorOpenDuration)
				fmt.Printf("reset dør")
				ce := ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				// fmt.Printf("[FSM %s] clear attempt floor=%d dir=%v orders=%+v ce=%+v\n", myID, e.Floor, e.TravelDir, ordersAtFloorSnapshot(&e.Orders, e.Floor), ce)
				clearCh <- ce
				continue
			}

			// Hvis idle i etasje og har ordre her: åpne
			if e.Behavior == EB_Idle && e.Floor >= 0 && om.HasOrderAtFloor(&e.Orders, e.Floor) {
				e.Behavior = EB_DoorOpen
				openDoorAndSetLamp(timer)
				//clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)

				ce := ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				// fmt.Printf("[FSM %s] clear attempt floor=%d dir=%v orders=%+v ce=%+v\n", myID, e.Floor, e.TravelDir, ordersAtFloorSnapshot(&e.Orders, e.Floor), ce)
				clearCh <- ce
				publishIfChanged()
				continue
			}

			if e.Behavior == EB_Idle {
				travelDir, behavior, dir := chooseDirection(&e)
				e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
				if e.Behavior == EB_Moving {
					setMotor(e.Dir)
				}
				publishIfChanged()
			}

		// -------- Floor sensor --------
		case floor := <-floorCh:
			e.Floor = floor
			elevio.SetFloorIndicator(floor)

			if e.Behavior == EB_Moving && !stopPressed {
				if shouldStop(&e) {
					stopMotor()
					e.Dir = elevio.MD_Stop
					e.Behavior = EB_Idle
					// fmt.Printf("[FSM %s] Got orders, behavior=%v, floor=%d, traveldir=%v, direction=%v\n", myID, e.Behavior, e.Floor, e.TravelDir, e.Dir)
					if om.HasOrderAtFloor(&e.Orders, e.Floor) {
						e.Behavior = EB_DoorOpen
						openDoorAndSetLamp(timer)
						ce := ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
						// fmt.Printf("[FSM %s] clear attempt floor=%d dir=%v orders=%+v ce=%+v\n", myID, e.Floor, e.TravelDir, ordersAtFloorSnapshot(&e.Orders, e.Floor), ce)
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
			publishIfChanged()

		// -------- Button light update --------
		case lightState := <-setButtonLight:
			updateButtonLights(lightState)

		// -------- Door timeout --------
		case <-timer.Timeout():
			if e.Behavior != EB_DoorOpen {
				continue
			}
			if obstructed || stopPressed {
				timer.Reset(doorOpenDuration)
				continue
			}
/* 
			// Hold doren apen og retry clear sa lenge ordre fortsatt finnes pa etasjen.
		 	if e.Floor >= 0 && om.HasOrderAtFloor(&e.Orders, e.Floor) {
				timer.Reset(doorOpenDuration)
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				continue
			}  */

			closeDoorAndResetLamp(timer)

			travelDir, behavior, dir := chooseDirection(&e)
			e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
			if e.Behavior == EB_Moving {
				setMotor(e.Dir)
			}

			publishIfChanged()

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
			publishIfChanged()
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

func updateButtonLights(ls config.LightState) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		elevio.SetButtonLamp(config.BT_HallUp, floor, ls.Hall[floor][config.BT_HallUp])
		elevio.SetButtonLamp(config.BT_HallDown, floor, ls.Hall[floor][config.BT_HallDown])
		elevio.SetButtonLamp(config.BT_Cab, floor, ls.Cab[floor])
	}
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

	/* ce.ClearCab = orders.Cab[floor]
	ce.ClearHallUp = orders.Hall[floor][config.BT_HallUp]
	ce.ClearHallDown = orders.Hall[floor][config.BT_HallDown] */

	hallUp := orders.Hall[floor][config.BT_HallUp]
	hallDown := orders.Hall[floor][config.BT_HallDown]

	ce.ClearCab = orders.Cab[floor]

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

		default:
		ce.ClearHallUp = hallUp
		ce.ClearHallDown = hallDown 
	}

	return ce
}


/* func ordersAtFloorSnapshot(o *om.Orders, floor int) string {
	if floor < 0 {
		return "invalid floor"
	}
	return fmt.Sprintf("cab=%v hallUp=%v hallDown=%v",
		o.Cab[floor],
		o.Hall[floor][config.BT_HallUp],
		o.Hall[floor][config.BT_HallDown],
	)
} */