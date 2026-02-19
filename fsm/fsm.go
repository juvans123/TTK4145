package fsm

import (
	"fmt"
	c "heis/config"
	e "heis/elevio"
	om "heis/ordermanagement"
	//t "heis/timer"
)

func Run(orders_Executed chan<- om.ExecutedOrder, ordersCmd <-chan om.Orders, doorTimeOut <-chan bool, floor_arrived <-chan int, obstruction_pressed <-chan bool, stopButton_pressed <-chan bool, timerCmd chan<- bool) {

	numFloors := 4

	CurrentOrders := om.Orders{
		Cab:  make([]bool, 4),
		Hall: [4][2]bool{},
	}

	var d e.MotorDirection
	var state State
	var travelDir c.TravelDirection

	initElevator()
	setButtonLights(CurrentOrders)
	//new order matrix
	for {
		select {
		case orders := <-ordersCmd:

			CurrentOrders = orders

			setButtonLights(CurrentOrders)

			switch state {
			case EB_Idle:
				if om.HasOrderAtFloor(CurrentOrders, currentFloor) {
					handleOrderAtCurrentFloor(CurrentOrders, &state, travelDir, timerCmd, orders_Executed)
				} else {
					SetDirectionAndState(travelDir, &state, CurrentOrders)
				}
			case EB_DoorOpen:
				// Hvis døren allerede er åpen på denne etasjen, restart timer og oppdater lys
				handleOrderAtCurrentFloor(CurrentOrders, &state, travelDir, timerCmd, orders_Executed)
			case EB_Moving:
				// ingenting spesial, heisen vil stoppe når den kommer til gulvet
			}

		case floor := <-floor_arrived:

			currentFloor = floor

			e.SetFloorIndicator(currentFloor)
			travelDir = GetTravelDirection(d)

			if ShouldStop(CurrentOrders, currentFloor, d) {
				StopElevator(&state)

				if om.HasOrderAtFloor(CurrentOrders, currentFloor) {

					//om.ClearCurrentFloor(&CurrentOrders, currentFloor,  travelDir)
					orders_Executed <- om.ExecutedOrder{Floor: currentFloor, TravelDir: travelDir}
					setButtonLights(CurrentOrders)
					OpenDoor(&state)
					//state = EB_DoorOpen
					timerCmd <- true
				}
				//break //?

			} else if currentFloor == 0 {
				ElevatorUp(&state)
			} else if currentFloor == numFloors-1 {
				ElevatorDown(&state)
			}

		case a := <-obstruction_pressed:
			fmt.Printf("obstructed")
			if a {
				e.SetMotorDirection(e.MD_Stop)

			} else {
				e.SetMotorDirection(d)
			}

		case <-doorTimeOut:

			e.SetDoorOpenLamp(false)

			if om.HasOrderAtFloor(CurrentOrders, currentFloor) {
				OpenDoor(&state)
				timerCmd <- true
			} else if ordersExist(CurrentOrders) {
				state = EB_Moving
				d = NextMove(CurrentOrders, currentFloor, travelDir)
				e.SetMotorDirection(d)
			} else {
				state = EB_Idle
			}

		case a := <-stopButton_pressed:
			fmt.Printf("%+v\n", a)
			/* for floor := 0; floor < numFloors; floor++ {
				for button := e.ButtonType(0); button < 3; button++ {
					e.SetButtonLamp(button, floor, true)

				}
			} */
		}
	}

}

func handleOrderAtCurrentFloor(CurrentOrders om.Orders, state *State, travelDir c.TravelDirection,
	timerCmd chan<- bool, orders_Executed chan<- om.ExecutedOrder) {

	if om.HasOrderAtFloor(CurrentOrders, currentFloor) {
		OpenDoor(state)
		timerCmd <- true
		orders_Executed <- om.ExecutedOrder{Floor: currentFloor, TravelDir: travelDir}
		setButtonLights(CurrentOrders)
	}
}

func GetTravelDirection(motordir e.MotorDirection) c.TravelDirection {
	switch motordir {
	case e.MD_Up:
		return c.TD_Up
	case e.MD_Down:
		return c.TD_Down
	default:
		return c.TD_Up
	}
}

func ClearLightsCurrentFloor(travelDir c.TravelDirection, orders om.Orders) {
	e.SetButtonLamp(e.BT_Cab, currentFloor, false)

	switch travelDir {
	case c.TD_Up:
		if om.OrdersAbove(orders, currentFloor) {
			e.SetButtonLamp(e.BT_HallUp, currentFloor, false)
		} else {
			e.SetButtonLamp(e.BT_HallDown, currentFloor, false)
			e.SetButtonLamp(e.BT_HallUp, currentFloor, false)
		}
	case c.TD_Down:
		if om.OrdersBelow(orders, currentFloor) {
			e.SetButtonLamp(e.BT_HallDown, currentFloor, false)
		} else {
			e.SetButtonLamp(e.BT_HallDown, currentFloor, false)
			e.SetButtonLamp(e.BT_HallUp, currentFloor, false)
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

func setButtonLights(orders om.Orders) {
	for floor := 0; floor < len(orders.Cab); floor++ {
		e.SetButtonLamp(e.BT_Cab, floor, orders.Cab[floor])
		e.SetButtonLamp(e.BT_HallUp, floor, orders.Hall[floor][e.BT_HallUp])
		e.SetButtonLamp(e.BT_HallDown, floor, orders.Hall[floor][e.BT_HallDown])
	}
}

func ElevatorDown(state *State) {
	e.SetMotorDirection(e.MD_Down)
	*state = EB_Moving
}

func ElevatorUp(state *State) {
	e.SetMotorDirection(e.MD_Up)
	*state = EB_Moving
}

func SetDirectionAndState(travelDir c.TravelDirection, state *State, orders om.Orders) {
	direction := NextMove(orders, currentFloor, travelDir)
	e.SetMotorDirection(direction)
	*state = EB_Moving
}

func StopElevator(state *State) {
	e.SetMotorDirection(e.MD_Stop)
	*state = EB_Idle
}

func OpenDoor(state *State) {
	e.SetDoorOpenLamp(true)
	*state = EB_DoorOpen

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

func initElevator() {
	e.SetDoorOpenLamp(false)
	e.SetMotorDirection(e.MD_Down)

}

/* func UpdateOrdersAndLamps(ButtonEvent e.ButtonEvent, ) {
	switch ButtonEvent.Button {
	case e.BT_Cab:
		CurrentOrders.Cab[ButtonEvent.Floor] = true
	case e.BT_HallUp:
		CurrentOrders.Hall[ButtonEvent.Floor][e.BT_HallUp] = true
	case e.BT_HallDown:
		CurrentOrders.Hall[ButtonEvent.Floor][e.BT_HallDown] = true
	}
	<- e.DriverCmd{Type: e.SetButtonLamp, Button: ButtonEvent.Button, Floor: ButtonEvent.Floor, Value: true}
}
*/

/*
func SetButtonLamps(ButtonEvent e.ButtonEvent, ){
	<- e.DriverCmd{Type: e.SetButtonLamp, Button: ButtonEvent.Button, Floor: ButtonEvent.Floor, Value: true}
} */

/*
	 func ClearAllOrders() {
		for floor := 0; floor < 4; floor++ {

			CurrentOrders.Cab[floor] = false
			CurrentOrders.Hall[floor][0] = false
			CurrentOrders.Hall[floor][1] = false

			for bt := e.ButtonType(0); bt < 3; bt++ {
				<- e.DriverCmd{Type: e.SetButtonLamp, Button: bt, Floor: floor, Value: false}
			}
		}
	}
*/

/* func ClearCurrentFloor(orders Orders, currentFloor int, , travelDir TravelDirection) {
	orders.Cab[currentFloor] = false
	<- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_Cab, Floor: currentFloor, Value: false}

	switch travelDir {
	case TD_Up:
		if OrdersAbove(orders, currentFloor) {
			CurrentOrders.Hall[currentFloor][e.BT_HallUp] = false
			<- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: currentFloor, Value: false}
		} else {
			CurrentOrders.Hall[currentFloor][e.BT_HallUp] = false
			CurrentOrders.Hall[currentFloor][e.BT_HallDown] = false
			<- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: currentFloor, Value: false}
			<- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: currentFloor, Value: false}
		}
	case TD_Down:
		if OrdersBelow(orders, currentFloor) {
			CurrentOrders.Hall[currentFloor][e.BT_HallDown] = false
			<- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: currentFloor, Value: false}
		} else {
			CurrentOrders.Hall[currentFloor][e.BT_HallUp] = false
			CurrentOrders.Hall[currentFloor][e.BT_HallDown] = false
			<- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallDown, Floor: currentFloor, Value: false}
			<- e.DriverCmd{Type: e.SetButtonLamp, Button: e.BT_HallUp, Floor: currentFloor, Value: false}
		}
	}

} */
