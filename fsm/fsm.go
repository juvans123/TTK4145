package fsm

import (
	//"fmt"
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

const (
	doorOpenDuration = 3 * time.Second
	motorImmobileTimeout = 3 * time.Second
	obstructionImmobileTimeout = 3 * time.Second
)

func Run(
	myID string,
	doorTimer DoorTimer,
	floorCh <-chan int,
	ordersFromOmCh <-chan om.Orders,
	obstrCh <-chan bool,
	stopButtonCh <-chan bool,
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

	lastPublishedState := PublicStateFromFSM(e, myID)
	tryPublishInitialState(stateOutCh, lastPublishedState)

	motorImmobileTimer, motorImmobileTimerActive := newStoppedTimer(motorImmobileTimeout)
	obstructionTimer, obstructionTimerActive := newStoppedTimer(obstructionImmobileTimeout)

	for {
		select {
		case newOrders := <-ordersFromOmCh:
			prevOrders := e.Orders
			prevAtFloor := (e.Floor >= 0) && om.HasOrderAtFloor(&prevOrders, e.Floor)

			e.Orders = newOrders

			if stopPressed {
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
				continue
			}

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
				if shouldTakeOrderInCurrentTravelDir(&e) && !prevAtFloor {
					doorTimer.Reset(doorOpenDuration)
					clearCh <-ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				}
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
				continue
			}

			if e.Behavior == EB_Idle && shouldTakeOrderInCurrentTravelDir(&e) {
				e.Behavior = EB_DoorOpen
				openDoorAndStartTimer(doorTimer)
				if e.Obstructed {
					startObstructionTimer(obstructionTimer, &obstructionTimerActive)
				}
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
				continue
			}

			if e.Behavior == EB_Idle && e.Floor >= 0 && om.HasOrderAtFloor(&e.Orders, e.Floor) {
				if !shouldTakeOrderInCurrentTravelDir(&e) && hasOppositeHallOrderAtFloor(&e) {
					e.TravelDir = oppositeTravelDirection(e.TravelDir)
				}
				if shouldTakeOrderInCurrentTravelDir(&e) {
					e.Behavior = EB_DoorOpen
					openDoorAndStartTimer(doorTimer)
					if e.Obstructed {
						startObstructionTimer(obstructionTimer, &obstructionTimerActive)
					}
					clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
					publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
					continue
				}
			}

			if e.Behavior == EB_Idle {
				travelDir, behavior, dir := chooseDirection(&e)
				e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
				if e.Behavior == EB_Moving {
					setMotor(e.Dir)
					startMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
				} 
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
			}

		case floor := <-floorCh:
			e.Floor = floor
			elevio.SetFloorIndicator(floor)

			if e.Immobile && !e.Obstructed{
				e.Immobile = false
				e.Behavior = EB_Moving
			}

			if e.Behavior == EB_Moving {
				startMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
			} 

			if e.Behavior == EB_Moving && !stopPressed && shouldStop(&e) {
				stopMotor()
				stopMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
				e.Dir = elevio.MD_Stop
				e.Behavior = EB_Idle

				if shouldTakeOrderInCurrentTravelDir(&e) {
					e.Behavior = EB_DoorOpen
					openDoorAndStartTimer(doorTimer)
					if e.Obstructed {
						startObstructionTimer(obstructionTimer, &obstructionTimerActive)
					}
					clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				} else if !hasOrdersInTravelDirection(&e) && hasOppositeHallOrderAtFloor(&e) {
					e.TravelDir = oppositeTravelDirection(e.TravelDir)
					e.Behavior = EB_DoorOpen
					openDoorAndStartTimer(doorTimer)
					clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				}
				
			}
			publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)

		case lightState := <-setButtonLight:
			updateButtonLights(lightState)

		case <-doorTimer.Timeout():
			if e.Behavior != EB_DoorOpen {
				continue
			}
			if e.Obstructed || stopPressed {
				doorTimer.Reset(doorOpenDuration)
				continue
			}

			if shouldTakeOrderInCurrentTravelDir(&e) {
				doorTimer.Reset(doorOpenDuration)
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				continue
			}

			if !hasOrdersInTravelDirection(&e) && hasOppositeHallOrderAtFloor(&e) {
				e.TravelDir = oppositeTravelDirection(e.TravelDir)
				openDoorAndStartTimer(doorTimer)
				if e.Obstructed {
					startObstructionTimer(obstructionTimer, &obstructionTimerActive)
				}
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
				continue
			}

			closeDoorAndStopTimer(doorTimer)
			travelDir, behavior, dir := chooseDirection(&e)
			e.TravelDir, e.Behavior, e.Dir = travelDir, behavior, dir
			if e.Behavior == EB_Moving {
				setMotor(e.Dir)
				startMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
			} 

			publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)


		case isObstructed := <-obstrCh:
			e.Obstructed = isObstructed
			if isObstructed {
				if e.Behavior == EB_DoorOpen {
					startObstructionTimer(obstructionTimer, &obstructionTimerActive)
				}
			} else {
				stopObstructionTimer(obstructionTimer, &obstructionTimerActive)
				e.Immobile = false
				if e.Behavior == EB_DoorOpen {
					doorTimer.Reset(doorOpenDuration)
				}
			}

			publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)


		case stopSignal := <-stopButtonCh:
			stopPressed = stopSignal
			elevio.SetStopLamp(stopSignal)

			if stopSignal {
				stopMotor()
				stopMotorTimer(motorImmobileTimer, &motorImmobileTimerActive)
				stopObstructionTimer(obstructionTimer, &obstructionTimerActive)

				e.Dir = elevio.MD_Stop
				e.Behavior = EB_Idle

				currentFloor := elevio.GetFloor()

				if currentFloor >= 0 {
					e.Behavior = EB_DoorOpen
					openDoorAndStartTimer(doorTimer)
					//clearer ikke her
				} 
			} else {
				if e.Behavior == EB_DoorOpen {
					doorTimer.Reset(doorOpenDuration)
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
			publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)

		case <-motorImmobileTimer.C:
			motorImmobileTimerActive = false
			if e.Behavior == EB_Moving && !stopPressed {
				e.Immobile = true
				e.Dir = elevio.MD_Stop
				e.Behavior = EB_Idle
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
			}
		
		case <-obstructionTimer.C:
			obstructionTimerActive = false
			if e.Obstructed && e.Behavior == EB_DoorOpen && !stopPressed {
				e.Immobile = true
				stopMotor()
				e.Dir = elevio.MD_Stop
				e.Behavior = EB_DoorOpen
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
			}
		}
	}
}


