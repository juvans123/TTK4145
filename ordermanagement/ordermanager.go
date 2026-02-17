package ordermanagement

import (
	c "heis/config"
	"heis/elevio"
)

// Run kjører order-managerens hovedloop
func Run(button_pressed <-chan elevio.ButtonEvent, ordersCmd chan<- Orders) {

	orders := Orders{
		Cab:  make([]bool, 4),
		Hall: [4][2]bool{},
	}

	for button := range button_pressed {
		UpdateOrders(button, &orders)
		ordersCmd <- orders
	}
}

// UpdateOrders oppdaterer ordrelisten basert på knappetrykk
func UpdateOrders(buttonEvent elevio.ButtonEvent, orders *Orders) {
	switch buttonEvent.Button {
	case elevio.BT_Cab:
		// Cab-knapp (lokal til denne elevatoren)
		orders.Cab[buttonEvent.Floor] = true

	case elevio.BT_HallUp:
		// Hall opp-knapp (delt med alle elevatorer)
		orders.Hall[buttonEvent.Floor][0] = true

	case elevio.BT_HallDown:
		// Hall ned-knapp (delt med alle elevatorer)
		orders.Hall[buttonEvent.Floor][1] = true
	}
}

func ClearAllOrders(elevioCmd chan<- elevio.DriverCmd, orders *Orders) {
	for floor := 0; floor < 4; floor++ {

		orders.Cab[floor] = false
		orders.Hall[floor][0] = false
		orders.Hall[floor][1] = false
	}
}

func ClearCurrentFloor(orders Orders, currentFloor int, elevioCmd chan<- elevio.DriverCmd, travelDir c.TravelDirection) {
	orders.Cab[currentFloor] = false
	//elevioCmd <- elevio.DriverCmd{Type: elevio.SetButtonLamp, Button: elevio.BT_Cab, Floor: currentFloor, Value: false}

	switch travelDir {
	case c.TD_Up:
		if OrdersAbove(orders, currentFloor) {
			orders.Hall[currentFloor][elevio.BT_HallUp] = false
			//elevioCmd <- elevio.DriverCmd{Type: elevio.SetButtonLamp, Button: elevio.BT_HallUp, Floor: currentFloor, Value: false}
		} else {
			orders.Hall[currentFloor][elevio.BT_HallUp] = false
			orders.Hall[currentFloor][elevio.BT_HallDown] = false
			//elevioCmd <- elevio.DriverCmd{Type: elevio.SetButtonLamp, Button: elevio.BT_HallDown, Floor: currentFloor, Value: false}
			//elevioCmd <- elevio.DriverCmd{Type: elevio.SetButtonLamp, Button: elevio.BT_HallUp, Floor: currentFloor, Value: false}
		}
	case c.TD_Down:
		if OrdersBelow(orders, currentFloor) {
			orders.Hall[currentFloor][elevio.BT_HallDown] = false
			//elevioCmd <- elevio.DriverCmd{Type: elevio.SetButtonLamp, Button: elevio.BT_HallDown, Floor: currentFloor, Value: false}
		} else {
			orders.Hall[currentFloor][elevio.BT_HallUp] = false
			orders.Hall[currentFloor][elevio.BT_HallDown] = false
			//elevioCmd <- elevio.DriverCmd{Type: elevio.SetButtonLamp, Button: elevio.BT_HallDown, Floor: currentFloor, Value: false}
			//elevioCmd <- elevio.DriverCmd{Type: elevio.SetButtonLamp, Button: elevio.BT_HallUp, Floor: currentFloor, Value: false}
		}
	}

}

func OrdersAbove(orders Orders, currentFloor int) bool {
	for floor := currentFloor + 1; floor < len(orders.Cab); floor++ {
		if HasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}

func OrdersBelow(orders Orders, currentFloor int) bool {
	for floor := currentFloor - 1; floor >= 0; floor-- {
		if HasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}

func HasOrderAtFloor(orders Orders, floor int) bool {
	return orders.Cab[floor] == true ||
		orders.Hall[floor][elevio.BT_HallUp] == true ||
		orders.Hall[floor][elevio.BT_HallDown] == true
}

/* type Channels struct {
	HallPressed <-chan HallButtonEvent
	CabPressed <-chan CabButtonEvent
	HallCleared <-chan ClearHallEvent
	CabCleared <-chan ClearCabEvent

	//PeerUpdate <-chan PeerUpdateWrap
	RemoteState <-chan ElevatorState

	OrdersOut chan<- Orders
}

// type PeerUpdateWrap

func Run(myID string, ch Channels){
	ws := WorldState{
		HallRequests: // FIKS DETTE
		ElevStates: map[string]ElevatorState{},
		Alive: map[string]bool{},
	}
	ws.Alive[myID] = true
	// mye greier her med triggerReassign

	ch.OrdersOut <- Orders{Cab: append([]bool(nil), myCab...)}

}
*/
