package ordermanagement

import (
	"heis/config"
)

const NumFloors = 4

func Run(
	buttonCh <- chan config.ButtonEvent, // fra elevio poller
	clearCh <- chan config.ClearEvent, // fra FSM når dør ordre er servert
	omOrdersCh chan <- Orders, // snapshot til FSM

){
	orders := NewOrders()
	// noe med publish ??
	for {
		select{
		case btn := <-buttonCh:
			addOrderFromButtonEvent(btn, &orders)
			omOrdersCh <- orders
			// publish ??
		case cl := <-clearCh:
			ClearAtFloor(&orders, cl.Floor, cl.Dir)
			omOrdersCh <- orders
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

func addOrderFromButtonEvent(btn config.ButtonEvent, orders *Orders) {
	switch btn.Button {
	case config.BT_Cab:
		orders.Cab[btn.Floor] = true

	case config.BT_HallUp:
		orders.Hall[btn.Floor][config.BT_HallUp] = true

	case config.BT_HallDown:
		orders.Hall[btn.Floor][config.BT_HallDown] = true
	}
}

func ClearAtFloor(orders *Orders, floor int, travelDir config.TravelDirection) {
	if floor >= 0 && floor < len(orders.Cab){
		orders.Cab[floor] = false
	}
	switch travelDir {
	case config.TD_Up:
		orders.Hall[floor][config.BT_HallUp] = false
		if !OrdersAbove(orders, floor) {
			orders.Hall[floor][config.BT_HallDown] = false
		} 
	case config.TD_Down:
		orders.Hall[floor][config.BT_HallDown] = false
		if !OrdersBelow(orders, floor) {
			orders.Hall[floor][config.BT_HallUp] = false
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
		orders.Hall[floor][config.BT_HallUp]||
		orders.Hall[floor][config.BT_HallDown]
}
