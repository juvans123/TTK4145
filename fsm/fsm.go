package fsm

import (
	"heis/elevio"
	om "heis/ordermanagement"
	"heis/types"
	"time"
)

const (
	doorOpenDuration           = 3 * time.Second
	motorImmobileTimeout       = 3 * time.Second
	obstructionImmobileTimeout = 4 * time.Second
)

func Run(
	myID string,
	doorTimer *time.Timer,
	floorCh <-chan int,
	ordersFromOmCh <-chan om.Orders,
	obstrCh <-chan bool,
	stopButtonCh <-chan bool,
	clearCh chan<- types.ClearEvent,
	stateOutCh chan<- types.ElevatorState,
	setButtonLight <-chan types.LightState,
) {
	e := Elevator{
		Floor:        -1,
		Dir:          elevio.MD_Down,
		TravelDir:    types.TD_Down,
		Behavior:     EB_Moving,
		Orders:       om.NewOrders(types.N_FLOORS),
		IsObstructed: false,
		IsImmobile:   false,
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
			prevAtFloor := om.HasOrderAtFloor(&prevOrders, e.Floor)

			e.Orders = newOrders

			if stopPressed {
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
				continue
			}

			lastKnownFloor := e.Floor
			isBetweenFloors := elevio.GetFloor() == -1
			if e.IsImmobile && om.HasOrderAtFloor(&e.Orders, lastKnownFloor) && isBetweenFloors {
				e.Behavior = EB_Moving
				setMotor(resumeTowardsLastKnownFloor(e.TravelDir))
			}

			if e.Behavior == EB_DoorOpen {
				if shouldTakeOrderInCurrentTravelDir(&e) && !prevAtFloor {
					resetTimer(doorTimer, doorOpenDuration)
					clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				}
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
				continue
			}

			if e.Behavior == EB_Idle && shouldTakeOrderInCurrentTravelDir(&e) {
				e.Behavior = EB_DoorOpen
				openDoorAndStartTimer(doorTimer)
				if e.IsObstructed {
					startObstructionTimer(obstructionTimer, &obstructionTimerActive)
				}
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
				continue
			}

			if e.Behavior == EB_Idle && om.HasOrderAtFloor(&e.Orders, e.Floor) {
				if !shouldTakeOrderInCurrentTravelDir(&e) && hasOppositeHallOrderAtFloor(&e) {
					e.TravelDir = oppositeTravelDirection(e.TravelDir)
				}
				if shouldTakeOrderInCurrentTravelDir(&e) {
					e.Behavior = EB_DoorOpen
					openDoorAndStartTimer(doorTimer)
					if e.IsObstructed {
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

			if e.IsImmobile && !e.IsObstructed {
				e.IsImmobile = false
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
					if e.IsObstructed {
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

		case <-doorTimer.C:
			if e.Behavior != EB_DoorOpen {
				continue
			}
			if e.IsObstructed || stopPressed {
				resetTimer(doorTimer, doorOpenDuration)
				continue
			}

			if shouldTakeOrderInCurrentTravelDir(&e) {
				resetTimer(doorTimer, doorOpenDuration)
				clearCh <- ComputeClearEvent(&e.Orders, e.Floor, e.TravelDir)
				continue
			}

			if !hasOrdersInTravelDirection(&e) && hasOppositeHallOrderAtFloor(&e) {
				e.TravelDir = oppositeTravelDirection(e.TravelDir)
				openDoorAndStartTimer(doorTimer)
				if e.IsObstructed {
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
			e.IsObstructed = isObstructed
			if isObstructed {
				if e.Behavior == EB_DoorOpen {
					startObstructionTimer(obstructionTimer, &obstructionTimerActive)
				}
			} else {
				stopObstructionTimer(obstructionTimer, &obstructionTimerActive)
				e.IsImmobile = false
				if e.Behavior == EB_DoorOpen {
					resetTimer(doorTimer, doorOpenDuration)
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
				}
			} else {
				if e.Behavior == EB_DoorOpen {
					resetTimer(doorTimer, doorOpenDuration)
					if e.IsObstructed {
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
				e.IsImmobile = true
				e.Dir = elevio.MD_Stop
				e.Behavior = EB_Idle
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
			}

		case <-obstructionTimer.C:
			obstructionTimerActive = false
			if e.IsObstructed && e.Behavior == EB_DoorOpen && !stopPressed {
				e.IsImmobile = true
				stopMotor()
				e.Dir = elevio.MD_Stop
				e.Behavior = EB_DoorOpen
				publishStateIfChanged(myID, e, stateOutCh, &lastPublishedState)
			}
		}
	}
}
