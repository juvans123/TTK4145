package fsm

import (
	"heis/config"
	"heis/elevio"
	om "heis/ordermanagement"
	"time"
)

/* func Run(button_pressed <-chan e.ButtonEvent, ordersCmd <-chan om.Orders, doorTimeOut <-chan bool, floor_arrived <-chan int, obstruction_pressed <-chan bool, stopButton_pressed <-chan bool, elevioCmd chan<- e.DriverCmd, timerCmd chan<- bool) {

	numFloors := 4

	CurrentOrders := om.Orders{
		Cab:  make([]bool, 4),
		Hall: [4][2]bool{},
	}

	var d e.MotorDirection
	var state State
	var travelDir c.TravelDirection

	initElevator(elevioCmd)
	//new order matrix
	for {
		select {
		case orders := <-ordersCmd:

			CurrentOrders = orders

			setButtonLights(CurrentOrders, elevioCmd)

			if state == EB_Idle {

				SetDirectionAndState(elevioCmd, travelDir, state, CurrentOrders)
			}

		case floor := <-floor_arrived:

			currentFloor = floor
			elevioCmd <- e.DriverCmd{Type: e.SetFloorIndicator, Floor: currentFloor}
			travelDir = GetTravelDirection(d)

			if ShouldStop(CurrentOrders, currentFloor, d) {
				state = StopElevator(elevioCmd)

				if om.HasOrderAtFloor(CurrentOrders, currentFloor) {

					om.ClearCurrentFloor(CurrentOrders, currentFloor, elevioCmd, travelDir)
					ClearLightsCurrentFloor(elevioCmd, travelDir, CurrentOrders)
					OpenDoor(state, elevioCmd)
					timerCmd <- true
				}
				break //?

			} else if currentFloor == 0 {
				ElevatorUp(elevioCmd, state)
			} else if currentFloor == numFloors-1 {
				ElevatorDown(elevioCmd, state)
			}

		case a := <-obstruction_pressed:
			fmt.Printf("obstructed")
			if a {
				elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: e.MD_Stop}

			} else {
				elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: d}
			}

		case a := <-doorTimeOut:
			elevioCmd <- e.DriverCmd{Type: e.SetDoorLamp, Value: a}

			if ordersExist(CurrentOrders) {
				state = EB_Moving
				d = NextMove(CurrentOrders, currentFloor, travelDir)
				elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: d}
			} else {
				state = EB_Idle
			}

		case a := <-stopButton_pressed:
			fmt.Printf("%+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := e.ButtonType(0); b < 3; b++ {
					elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: b, Floor: f, Value: true}

				}
			}
		}
	}

}
 */
 type DoorTimer interface {
	Reset(d time.Duration)
	Stop()
	Timeout() <-chan struct{}
}
const doorOpenDuration = 3 * time.Second

/* func Run(
	elevioCmd chan<- elevio.DriverCmd,
    floorCh <-chan int,             
    btnCh <-chan elevio.ButtonEvent,            
    obstructionCh <-chan bool,       
    stopButtonCh <-chan bool,    
	doorTimeoutCh <- chan struct{},
	doorTimerResetCh chan <- time.Duration, 
){

} */

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

func openDoor(t DoorTimer) {
	elevio.SetDoorOpenLamp(true)
	t.Reset(doorOpenDuration)
}

func closeDoor(t DoorTimer) {
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
		if e.Orders.Hall[floor][elevio.BT_HallUp]{
			return true
		}
		if e.Orders.Hall[floor][elevio.BT_HallDown] && !om.OrdersAbove(&e.Orders, floor){
			return true
		}
	case config.TD_Down:
		if e.Orders.Hall[floor][elevio.BT_HallDown]{
			return true
		}
		if e.Orders.Hall[floor][elevio.BT_HallUp] && !om.OrdersBelow(&e.Orders, floor){
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


func Run(
	timer DoorTimer,
	floorCh <-chan int,
	ordersCh <-chan om.Orders,
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
		case newOrders := <-ordersCh:
			e.Orders = newOrders

			if stopPressed {
				// Ikke start å kjøre når stopp er inne
				continue
			}

			// Hvis vi er idle og står i etasje og har ordre her: åpne dør direkte
			if e.Behavior == EB_Idle && hasOrderAtThisFloor(&e) {
				// velg “service retning” basert på dagens dir (brukes i ClearAtFloor-policy)
				if e.Floor >= 0 {
					e.Behavior = EB_DoorOpen
					openDoor(timer)
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
					openDoor(timer)

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

			closeDoor(timer)
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
					openDoor(timer)
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

/* func GetTravelDirection(motordir e.MotorDirection) c.TravelDirection {
	switch motordir {
	case e.MD_Up:
		return c.TD_Up
	case e.MD_Down:
		return c.TD_Down
	default:
		return c.TD_Up
	}
}

func ClearLightsCurrentFloor(elevioCmd chan<- e.DriverCmd, travelDir c.TravelDirection, orders om.Orders) {

	elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_Cab, Floor: currentFloor, Value: false}

	switch travelDir {
	case c.TD_Up:
		if om.OrdersAbove(orders, currentFloor) {
			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: currentFloor, Value: false}
		} else {

			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: currentFloor, Value: false}
			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: currentFloor, Value: false}
		}
	case c.TD_Down:
		if om.OrdersBelow(orders, currentFloor) {

			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: currentFloor, Value: false}
		} else {

			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: currentFloor, Value: false}
			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: currentFloor, Value: false}
		}
	}
}

func ShouldStop(orders om.Orders, currentFloor int, dir e.MotorDirection) bool {
	if orders.Cab[currentFloor] {
		return true
	}
	if orders.Hall[currentFloor][e.BT_HallUp] && dir == e.MD_Up {
		return true
	}
	if orders.Hall[currentFloor][e.BT_HallUp] && !om.OrdersBelow(orders, currentFloor) {
		return true
	}
	if orders.Hall[currentFloor][e.BT_HallDown] && dir == e.MD_Down {
		return true
	}
	if orders.Hall[currentFloor][e.BT_HallDown] && !om.OrdersAbove(orders, currentFloor) {
		return true
	}
	if !ordersExist(orders) {
		return true
	}

	return false
}


func setButtonLights(orders om.Orders, elevioCmd chan<- e.DriverCmd) {
	for floor := 0; floor < len(orders.Cab); floor++ {
		elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_Cab, Floor: floor, Value: orders.Cab[floor]}
		elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: floor, Value: orders.Hall[floor][e.BT_HallUp]}
		elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: floor, Value: orders.Hall[floor][e.BT_HallDown]}
	}
}

func ElevatorDown(elevioCmd chan<- e.DriverCmd) {
	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: e.MD_Down}
}

func ElevatorUp(elevioCmd chan<- e.DriverCmd) {
	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: e.MD_Up}
}

func SetDirectionAndState(elevioCmd chan<- e.DriverCmd, travelDir c.TravelDirection, orders om.Orders) {
	direction := NextMove(orders, currentFloor, travelDir)
	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: direction}
}

func StopElevator(elevioCmd chan<- e.DriverCmd){
	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: e.MD_Stop}
}

func OpenDoor(state State, elevioCmd chan<- e.DriverCmd) {
	elevioCmd <- e.DriverCmd{Type: e.SetDoorLamp, Value: true}
	state = EB_DoorOpen

}

func NextMove(orders om.Orders, currentFloor int, travelDir c.TravelDirection) e.MotorDirection {
	if om.HasOrderAtFloor(orders, currentFloor) || !ordersExist(orders) {
		return e.MD_Stop
	}
	above := om.OrdersAbove(orders, currentFloor)
	below := om.OrdersBelow(orders, currentFloor)

	switch travelDir {
	case c.TD_Up:
		if above {
			return e.MD_Up
		} else {
			return e.MD_Down
		}
	case c.TD_Down:
		if below {
			return e.MD_Down
		} else {
			return e.MD_Up
		}
	}
	return e.MD_Stop
}

func ordersExist(orders om.Orders) bool {
	for floor := 0; floor < len(orders.Cab); floor++ {
		if om.HasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}

func initElevator(elevioCmd chan<- e.DriverCmd) {
	elevioCmd <- e.DriverCmd{Type: e.SetDoorLamp, Value: false}
	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: e.MD_Down}

}
 */
/* func ClearCurrentFloor(orders Orders, currentFloor int, elevioCmd chan<- e.DriverCmd, travelDir TravelDirection) {
	orders.Cab[currentFloor] = false
	elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_Cab, Floor: currentFloor, Value: false}

	switch travelDir {
	case TD_Up:
		if OrdersAbove(orders, currentFloor) {
			CurrentOrders.Hall[currentFloor][e.BT_HallUp] = false
			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: currentFloor, Value: false}
		} else {
			CurrentOrders.Hall[currentFloor][e.BT_HallUp] = false
			CurrentOrders.Hall[currentFloor][e.BT_HallDown] = false
			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: currentFloor, Value: false}
			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: currentFloor, Value: false}
		}
	case TD_Down:
		if OrdersBelow(orders, currentFloor) {
			CurrentOrders.Hall[currentFloor][e.BT_HallDown] = false
			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: currentFloor, Value: false}
		} else {
			CurrentOrders.Hall[currentFloor][e.BT_HallUp] = false
			CurrentOrders.Hall[currentFloor][e.BT_HallDown] = false
			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: currentFloor, Value: false}
			elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: currentFloor, Value: false}
		}
	}

} */


/* func UpdateOrdersAndLamps(ButtonEvent e.ButtonEvent, elevioCmd chan<- e.DriverCmd) {
	switch ButtonEvent.Button {
	case e.BT_Cab:
		CurrentOrders.Cab[ButtonEvent.Floor] = true
	case e.BT_HallUp:
		CurrentOrders.Hall[ButtonEvent.Floor][e.BT_HallUp] = true
	case e.BT_HallDown:
		CurrentOrders.Hall[ButtonEvent.Floor][e.BT_HallDown] = true
	}
	elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: ButtonEvent.Button, Floor: ButtonEvent.Floor, Value: true}
}
*/

/*
func SetButtonLamps(ButtonEvent e.ButtonEvent, elevioCmd chan<- e.DriverCmd){
	elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: ButtonEvent.Button, Floor: ButtonEvent.Floor, Value: true}
} */

/*
	 func ClearAllOrders(elevioCmd chan<- e.DriverCmd) {
		for floor := 0; floor < 4; floor++ {

			CurrentOrders.Cab[floor] = false
			CurrentOrders.Hall[floor][0] = false
			CurrentOrders.Hall[floor][1] = false

			for bt := e.ButtonType(0); bt < 3; bt++ {
				elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: bt, Floor: floor, Value: false}
			}
		}
	}
*/

