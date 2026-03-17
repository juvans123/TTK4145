package fsm

import (
	//"fmt"
	"heis/config"
	om "heis/ordermanagement"
)

func shouldTakeOrderInCurrentTravelDir(e *Elevator) bool {
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


