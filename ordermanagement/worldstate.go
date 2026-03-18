package ordermanagement

import (
	types "heis/types"
)
func NewWorldState() WorldState {
	return WorldState{
		ConfirmedHallOrders: [types.N_FLOORS][2]bool{},
		ConfirmedCabOrders:  make(map[string][]bool),
		States:              make(map[string]types.ElevatorState),
		Alive:               make(map[string]bool),
	}
}


 func isOrderConfirmedInWorldState(ws *WorldState, key OrderKey) bool {
	switch key.Button {
	case types.BT_Cab:
		cabOrders, ok := ws.ConfirmedCabOrders[key.OwnerID]
		if !ok {
			return false
		}
		return cabOrders[key.Floor]

	case types.BT_HallUp, types.BT_HallDown:
		return ws.ConfirmedHallOrders[key.Floor][key.Button]
	}

	return false
}

func confirmOrderInWorldState(ws *WorldState, key OrderKey) {
	switch key.Button {
	case types.BT_Cab:
		if _, ok := ws.ConfirmedCabOrders[key.OwnerID]; !ok {
			ws.ConfirmedCabOrders[key.OwnerID] = make([]bool, types.N_FLOORS)
		}
		ws.ConfirmedCabOrders[key.OwnerID][key.Floor] = true

	case types.BT_HallUp, types.BT_HallDown:
		ws.ConfirmedHallOrders[key.Floor][key.Button] = true
	}
}

func clearOrderInWorldState(ws *WorldState, key OrderKey) {
	switch key.Button {
	case types.BT_Cab:
		if cabs, ok := ws.ConfirmedCabOrders[key.OwnerID]; ok {
			cabs[key.Floor] = false
		}

	case types.BT_HallUp, types.BT_HallDown:
		ws.ConfirmedHallOrders[key.Floor][key.Button] = false
	}
}