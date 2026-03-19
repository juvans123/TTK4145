package ordermanagement

import ("elevator/types")

func OrdersAbove(orders *Orders, currentFloor int) bool {
	for floor := currentFloor + 1; floor < types.N_FLOORS; floor++ {
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
		orders.Hall[floor][types.BT_HallUp] ||
		orders.Hall[floor][types.BT_HallDown]
}

func HasOrders(orders *Orders) bool {
	for floor := 0; floor < types.N_FLOORS; floor++ {
		if HasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}
