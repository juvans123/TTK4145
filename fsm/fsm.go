package fsm


import (
	"fmt"
	e "heis/elevio"
)

func Run(button_pressed <-chan e.ButtonEvent, doorTimeOut <-chan bool, floor_arrived <-chan int, obstruction_pressed <-chan bool, stopButton_pressed <-chan bool, elevioCmd chan<- e.DriverCmd, timerCmd chan<- bool) {

	numFloors := 4
	var d e.MotorDirection
	var state State
	var travelDir TravelDirection

	ClearAllOrders(elevioCmd)
	initElevator(elevioCmd)

	for {
		select {
		case button := <-button_pressed:

			UpdateOrdersAndLamps(button, elevioCmd)
			
			if state == EB_Idle {


				SetDirectionAndState(elevioCmd, travelDir, state)
			}

		case floor := <-floor_arrived:
		
			currentFloor = floor
			elevioCmd <- e.DriverCmd{Type: e.SetFloorIndicator, Floor: currentFloor}
			travelDir = GetTravelDirection(d)

			if ShouldStop(CurrentOrders, currentFloor, d) {
				StopElevator(state, elevioCmd)

				if hasOrderAtFloor(CurrentOrders, currentFloor) {

					ClearCurrentFloor(CurrentOrders, currentFloor, elevioCmd, travelDir)
					OpenDoor(state, elevioCmd)
					timerCmd <- true
				}
				break  //chat melder at denne ikke gjør noe

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

func GetTravelDirection(motordir e.MotorDirection) TravelDirection {
	switch motordir {
	case e.MD_Up:
		return TD_Up
	case e.MD_Down:
		return TD_Down
	default:
		return TD_Up
	}
}

func ShouldStop(orders Orders, currentFloor int, dir e.MotorDirection) bool {
	if orders.Cab[currentFloor] {
		return true
	}
	if orders.Hall[currentFloor][e.BT_HallUp] && dir == e.MD_Up {
		return true
	}
	if orders.Hall[currentFloor][e.BT_HallUp] && !OrdersBelow(orders, currentFloor) {
		return true
	}
	if orders.Hall[currentFloor][e.BT_HallDown] && dir == e.MD_Down {
		return true
	}
	if orders.Hall[currentFloor][e.BT_HallDown] && !OrdersAbove(orders, currentFloor) {
		return true
	}
	if !ordersExist(orders) {
		return true
	}

	return false
}

func ClearCurrentFloor(orders Orders, currentFloor int, elevioCmd chan<- e.DriverCmd, travelDir TravelDirection) {
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

}

func ElevatorDown(elevioCmd chan<- e.DriverCmd, state State) {
	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: e.MD_Down}
	state = EB_Moving
}

func ElevatorUp(elevioCmd chan<- e.DriverCmd, state State) {
	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: e.MD_Up}
	state = EB_Moving
}

func SetDirectionAndState(elevioCmd chan<- e.DriverCmd, travelDir TravelDirection, state State) {
	direction := NextMove(CurrentOrders, currentFloor, travelDir)
	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: direction}
	state = EB_Moving
} 

func StopElevator(state State, elevioCmd chan<- e.DriverCmd) {

	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: e.MD_Stop}
	state = EB_Idle
}

func OpenDoor(state State, elevioCmd chan<- e.DriverCmd) {
	elevioCmd <- e.DriverCmd{Type: e.SetDoorLamp, Value: true}
	state = EB_DoorOpen

}

func NextMove(orders Orders, currentFloor int, travelDir TravelDirection) e.MotorDirection {
	if hasOrderAtFloor(orders, currentFloor) || !ordersExist(orders) {
		return e.MD_Stop
	}
	above := OrdersAbove(orders, currentFloor)
	below := OrdersBelow(orders, currentFloor)

	switch travelDir {
	case TD_Up:
		if above {
			return e.MD_Up
		} else {
			return e.MD_Down
		}
	case TD_Down:
		if below {
			return e.MD_Down
		} else {
			return e.MD_Up
		}
	}
	return e.MD_Stop
}

func hasOrderAtFloor(orders Orders, floor int) bool {
	return orders.Cab[floor] == true ||
		orders.Hall[floor][e.BT_HallUp] == true ||
		orders.Hall[floor][e.BT_HallDown] == true
}

func OrdersAbove(orders Orders, currentFloor int) bool {
	for floor := currentFloor + 1; floor < len(orders.Cab); floor++ {
		if hasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}

func OrdersBelow(orders Orders, currentFloor int) bool {
	for floor := currentFloor - 1; floor >= 0; floor-- {
		if hasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}

func ordersExist(orders Orders) bool {
	for floor := 0; floor < len(orders.Cab); floor++ {
		if hasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}

func UpdateOrdersAndLamps(ButtonEvent e.ButtonEvent, elevioCmd chan<- e.DriverCmd) {
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



/* 
func SetButtonLamps(ButtonEvent e.ButtonEvent, elevioCmd chan<- e.DriverCmd){
	elevioCmd <- e.DriverCmd{Type: e.SetButtonLamp, Button: ButtonEvent.Button, Floor: ButtonEvent.Floor, Value: true}
} */

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

func initElevator(elevioCmd chan<- e.DriverCmd) {
	elevioCmd <- e.DriverCmd{Type: e.SetDoorLamp, Value: false}
	elevioCmd <- e.DriverCmd{Type: e.SetMotorDirection, MotorDir: e.MD_Down}

}
