package ordermanagement

import (
	"heis/config"
	"heis/elevio"
)

const NumFloors = 4

func Run(
	buttonCh <- chan elevio.ButtonEvent, // fra elevio poller
	clearCh <- chan config.ClearEvent, // fra FSM når dør ordre er servert
	ordersCh chan <- Orders, // snapshot til FSM/lys
){
	orders := NewOrders()
	// noe med publish ??
	for {
		select{
		case btn := <-buttonCh:
			addOrderFromButtonEvent(btn, &orders)
			ordersCh <- orders
			// publish ??
		case cl := <-clearCh:
			ClearAtFloor(&orders, cl.Floor, cl.Dir)
			ordersCh <- orders
			// publish ??
		}
	}
}

func NewOrders() Orders {
	o := Orders{
		Cab:  make([]bool, NumFloors),
		Hall: make([][]bool, NumFloors),
	}

	for i := 0; i < NumFloors; i++ {
		o.Hall[i] = make([]bool, 3)
	}

	return o
}

// UpdateOrders oppdaterer ordrelisten basert på knappetrykk
func addOrderFromButtonEvent(btn elevio.ButtonEvent, orders *Orders) {
	switch btn.Button {
	case elevio.BT_Cab:
		// Cab-knapp (lokal til denne elevatoren)
		orders.Cab[btn.Floor] = true

	case elevio.BT_HallUp:
		// Hall opp-knapp (delt med alle elevatorer)
		orders.Hall[btn.Floor][elevio.BT_HallUp] = true

	case elevio.BT_HallDown:
		// Hall ned-knapp (delt med alle elevatorer)
		orders.Hall[btn.Floor][elevio.BT_HallDown] = true
	}
}

/* func clearAllOrders(orders *Orders) {
	for floor := 0; floor < 4; floor++ {

		orders.Cab[floor] = false
		orders.Hall[floor][0] = false
		orders.Hall[floor][1] = false
	}
} */

func ClearAtFloor(orders *Orders, floor int, travelDir config.TravelDirection) {
	//elevioCmd <- elevio.DriverCmd{Type: elevio.SetButtonLamp, Button: elevio.BT_Cab, Floor: currentFloor, Value: false}
	if floor >= 0 && floor < len(orders.Cab){
		orders.Cab[floor] = false
	}
	switch travelDir {
	case config.TD_Up:
		orders.Hall[floor][elevio.BT_HallUp] = false
		if !OrdersAbove(orders, floor) {
			orders.Hall[floor][elevio.BT_HallDown] = false
		} 
	case config.TD_Down:
		orders.Hall[floor][elevio.BT_HallDown] = false
		if !OrdersBelow(orders, floor) {
			orders.Hall[floor][elevio.BT_HallUp] = false
		}
	}

}

func OrdersAbove(orders *Orders, currentFloor int) bool {
	for floor := currentFloor + 1; floor < len(orders.Cab); floor++ {
		if HasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}

func OrdersBelow(orders *Orders, currentFloor int) bool {
	for floor := currentFloor - 1; floor >= 0; floor-- {
		if HasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}

func HasOrderAtFloor(orders *Orders, floor int) bool {
	return orders.Cab[floor] ||
		orders.Hall[floor][elevio.BT_HallUp]||
		orders.Hall[floor][elevio.BT_HallDown]
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
