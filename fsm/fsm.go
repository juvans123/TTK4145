package fsm

import (
	//"fmt"
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
const motorImmobileTimeout = 3 * time.Second
const obstructionImmobileTimeout = 5 * time.Second

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
		Floor:      -1,
		Dir:        elevio.MD_Down,
		TravelDir:  config.TD_Down,
		Behavior:   EB_Moving,
		Orders:     om.NewOrders(4),
		Obstructed: false,
		Immobile:   false,
	}

	stopPressed := false
	elevatorInit(&e)

	lastPublished := PublicStateFromFSM(e, myID)
	select {
	case stateOutCh <- lastPublished:
	default:
	}

	motorImmobileTimer := time.NewTimer(motorImmobileTimeout)
	if !motorImmobileTimer.Stop() {
		select {
		case <-motorImmobileTimer.C:
		default:
		}
	}
	motorImmobileTimerActive := false

	obstructionTimer := time.NewTimer(obstructionImmobileTimeout)
	if !obstructionTimer.Stop() {
		select {
		case <-obstructionTimer.C:
		default:
		}
	}
	obstructionTimerActive := false

	publishIfChanged := func() {
		st := PublicStateFromFSM(e, myID)
		if st.Floor != lastPublished.Floor ||
			st.Behaviour != lastPublished.Behaviour ||
			st.Direction != lastPublished.Direction ||
			st.Obstructed != lastPublished.Obstructed ||
			st.Immobile != lastPublished.Immobile ||
			!cabRequestsEqual(st.CabRequests, lastPublished.CabRequests) {
			select {
			case stateOutCh <- st:
				fmt.Printf("[FSM %s] PUBLISH state floor=%d beh=%v dir=%v cab=%v\n", myID, st.Floor, st.Behaviour, st.Direction, st.CabRequests)
				lastPublished = st
			default:
			}
		}
	}

	for {
		select {

		// -------- Orders snapshot fra OM --------
		/* case newOrders := <-omOrdersCh:
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
		} */

		case newOrders := <-omOrdersCh:
			prevOrders := e.Orders
			prevAtFloor := (e.Floor >= 0) && om.HasOrderAtFloor(&prevOrders, e.Floor)

			e.Orders = newOrders

			if stopPressed {
				publishIfChanged()
				continue
			}

			//nowAtFloor := (e.Floor >= 0) && om.HasOrderAtFloor(&e.Orders, e.Floor)

			if e.Immobile && om.HasOrderAtFloor(&e.Orders, e.Floor) && elevio.GetFloor() == -1 {
				e.Behavior = EB_Moving
				switch e.TravelDir {
				case config.TD_Up:
					setMotor(elevio.MD_Down)
				case config.TD_Down:
					setMotor(elevio.MD_Up)
				}

			}

			if e.Behavior == EB_DoorOpen && e.Floor >= 0 {
				if shouldTakeOrdersAtFloor(&e) && !prevAtFloor {
					timer.Reset(doorOpenDuration)
					ce := ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
					clearCh <- ce
				}
				publishIfChanged()
				continue
			}

			if e.Behavior == EB_Idle && shouldTakeOrdersAtFloor(&e) {
				e.Behavior = EB_DoorOpen
				openDoorAndSetLamp(timer)
				if e.Obstructed {
					startObstructionTimer(obstructionTimer, &obstructionTimerActive)
				}
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				publishIfChanged()
				continue
			}

			if e.Behavior == EB_Idle && e.Floor >= 0 && om.HasOrderAtFloor(&e.Orders, e.Floor) {
				// Idle at floor with only opposite hall call: flip service direction so we can serve it.
				if !shouldTakeOrdersAtFloor(&e) && hasOppositeHallOrderAtFloor(&e) {
					e.TravelDir = oppositeTravelDirection(e.TravelDir)
				}
				if shouldTakeOrdersAtFloor(&e) {
					e.Behavior = EB_DoorOpen
					openDoorAndSetLamp(timer)
					if e.Obstructed {
						startObstructionTimer(obstructionTimer, &obstructionTimerActive)
					}
					clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
					publishIfChanged()
					continue
				}
			}

			if e.Behavior == EB_Idle {
				fmt.Printf("Behavior before chooseDir: %v\n", e.Behavior)
				travelDir, behavior, dir := chooseDirection(&e)
				fmt.Printf("Behavior after chooseDir: %v\n", e.Behavior)
				e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
				if e.Behavior == EB_Moving {
					setMotor(e.Dir)
					startMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
				} 
				publishIfChanged()
			}

		// -------- Floor sensor --------
		case floor := <-floorCh:
			e.Floor = floor
			elevio.SetFloorIndicator(floor)

			if e.Immobile && !e.Obstructed{
				e.Immobile = false
			}

			if e.Behavior == EB_Moving {
				startMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
			} 

			if e.Behavior == EB_Moving && !stopPressed {
				if shouldStop(&e) {
					stopMotor()
					stopMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
					e.Dir = elevio.MD_Stop
					e.Behavior = EB_Idle
					// fmt.Printf("[FSM %s] Got orders, behavior=%v, floor=%d, traveldir=%v, direction=%v\n", myID, e.Behavior, e.Floor, e.TravelDir, e.Dir)
					if om.HasOrderAtFloor(&e.Orders, e.Floor) {
						e.Behavior = EB_DoorOpen
						openDoorAndSetLamp(timer)
						if e.Obstructed {
							startObstructionTimer(obstructionTimer, &obstructionTimerActive)
						}
						ce := ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
						// fmt.Printf("[FSM %s] clear attempt floor=%d dir=%v orders=%+v ce=%+v\n", myID, e.Floor, e.TravelDir, ordersAtFloorSnapshot(&e.Orders, e.Floor), ce)
						clearCh <- ce
					} else if !hasOrdersInTravelDirection(&e) && hasOppositeHallOrderAtFloor(&e) {
						e.TravelDir = oppositeTravelDirection(e.TravelDir)
						e.Behavior = EB_DoorOpen
						openDoorAndSetLamp(timer)
						clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
					}
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
			if e.Obstructed || stopPressed {
				timer.Reset(doorOpenDuration)
				continue
			}

			// Hold doren apen kun hvis det finnes ordre pa etasjen som faktisk skal tas i valgt retning.
			if shouldTakeOrdersAtFloor(&e) {
				timer.Reset(doorOpenDuration)
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				continue
			}

			// Two-step clearing: if we are done in current travel direction at this floor,
			// flip direction and clear opposite hall call in a new door interval.
			if !hasOrdersInTravelDirection(&e) && hasOppositeHallOrderAtFloor(&e) {
				e.TravelDir = oppositeTravelDirection(e.TravelDir)
				openDoorAndSetLamp(timer)
				if e.Obstructed {
					startObstructionTimer(obstructionTimer, &obstructionTimerActive)
				}
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				publishIfChanged()
				continue
			}

			closeDoorAndResetLamp(timer)

			travelDir, behavior, dir := chooseDirection(&e)
			e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
			if e.Behavior == EB_Moving {
				setMotor(e.Dir)
				startMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
			} 

			publishIfChanged()

		// -------- Obstruction --------
		case obs := <-obstrCh:
			e.Obstructed = obs
			if obs {
				if e.Behavior == EB_DoorOpen {
					startObstructionTimer(obstructionTimer, &obstructionTimerActive)
				}
			} else {
				stopObstructionTimer(obstructionTimer, &obstructionTimerActive)
				e.Immobile = false
				if e.Behavior == EB_DoorOpen {
					timer.Reset(doorOpenDuration)
				}
			}

			publishIfChanged()

		// -------- Stop button --------
		case sp := <-stopCh:
			stopPressed = sp
			elevio.SetStopLamp(sp)

			if sp {
				stopMotor()
				stopMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
				stopObstructionTimer(obstructionTimer, &obstructionTimerActive)
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
					if e.Obstructed {
						startObstructionTimer(obstructionTimer, &obstructionTimerActive)
					}
				} else if e.Behavior == EB_Idle {

					travelDir, behavior, dir := chooseDirection(&e)
					e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
					if e.Behavior == EB_Moving {
						setMotor(e.Dir)
						startMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
					}
				}
			}
			publishIfChanged()
		case <-motorImmobileTimer.C:
			motorImmobileTimerActive = false
			if e.Behavior == EB_Moving && !stopPressed {
				e.Immobile = true
				//stopMotor()
				e.Dir = elevio.MD_Stop
				e.Behavior = EB_Idle
				publishIfChanged()
			}
		
		case <-obstructionTimer.C:
			obstructionTimerActive = false
			if e.Obstructed && e.Behavior == EB_DoorOpen && !stopPressed {
				e.Immobile = true
				stopMotor()
				e.Dir = elevio.MD_Stop
				e.Behavior = EB_DoorOpen
				publishIfChanged()
			}
		}
	}
}

/* func elevatorInit(e *Elevator) {
	//updateButtonLights(e)
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
	}
} */

func elevatorInit(e *Elevator) {
	// Hvis vi allerede står i en etasje, bare initialiser state riktig
	elevio.SetDoorOpenLamp(false)
	for floor := 0; floor < config.N_FLOORS; floor++ {
		elevio.SetButtonLamp(config.BT_HallUp, floor, false)
		elevio.SetButtonLamp(config.BT_HallDown, floor, false)
		elevio.SetButtonLamp(config.BT_Cab, floor, false)
	}
	
	if floor := elevio.GetFloor(); floor >= 0 {
		e.Floor = floor
		elevio.SetFloorIndicator(floor)
		e.Dir = elevio.MD_Stop
		e.Behavior = EB_Idle
		return
	}

	// Hvis vi er mellom etasjer: kjør ned til vi treffer en etasje
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

	// At end floors there is only one valid hall direction; clear it immediately
	// together with cab, regardless of current travel direction.
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

func shouldTakeOrdersAtFloor(e *Elevator) bool {
	if e.Floor < 0 || e.Floor >= len(e.Orders.Cab) {
		return false
	}
	ce := ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
	return ce.ClearCab || ce.ClearHallUp || ce.ClearHallDown
}

func oppositeTravelDirection(dir config.TravelDirection) config.TravelDirection {
	if dir == config.TD_Up {
		return config.TD_Down
	}
	return config.TD_Up
}

func hasOrdersInTravelDirection(e *Elevator) bool {
	if e.Floor < 0 {
		return false
	}
	if e.TravelDir == config.TD_Up {
		return om.OrdersAbove(&e.Orders, e.Floor)
	}
	return om.OrdersBelow(&e.Orders, e.Floor)
}

func hasOppositeHallOrderAtFloor(e *Elevator) bool {
	if e.Floor < 0 || e.Floor >= len(e.Orders.Hall) {
		return false
	}
	if e.TravelDir == config.TD_Up {
		return e.Orders.Hall[e.Floor][config.BT_HallDown]
	}
	return e.Orders.Hall[e.Floor][config.BT_HallUp]
}

func cabRequestsEqual(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func startTimer(timer *time.Timer, active *bool, timeOut time.Duration) {
	if *active {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}

	timer.Reset(timeOut)
	*active = true
}

func stopTimer(timer *time.Timer, active *bool) {
	if *active {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}

	*active = false
}



func startMotorTimer(timer *time.Timer, active *bool) {
	startTimer(timer, active, motorImmobileTimeout)
}

func stopMotorTimer(timer *time.Timer, active *bool) {
	stopTimer(timer, active)
}

func startObstructionTimer(timer *time.Timer, active *bool) {
	startTimer(timer, active, obstructionImmobileTimeout)
}

func stopObstructionTimer(timer *time.Timer, active *bool) {
	stopTimer(timer, active)
}