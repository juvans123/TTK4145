package ordermanagement

import (
	"heis/config"
)
func NewWorldState() WorldState {
	return WorldState{
		ConfirmedHallOrders: [config.N_FLOORS][2]bool{},
		ConfirmedCabOrders:  make(map[string][]bool),
		States:              make(map[string]config.ElevatorState),
		Alive:               make(map[string]bool),
	}
}
/* 
func confirmOrderInWorldState(ws *WorldState, key OrderKey) {

	switch key.Button {
	case config.BT_Cab:
		if _, ok := ws.ConfirmedCabOrders[key.OwnerID]; !ok {
			ws.ConfirmedCabOrders[key.OwnerID] = make([]bool, config.N_FLOORS)
		}
		if !ws.ConfirmedCabOrders[key.OwnerID][key.Floor] {
			ws.ConfirmedCabOrders[key.OwnerID][key.Floor] = true
			
		}

	case config.BT_HallUp, config.BT_HallDown:
		if !ws.ConfirmedHallOrders[key.Floor][key.Button] {
			ws.ConfirmedHallOrders[key.Floor][key.Button] = true
			
		}
	}
}

func clearOrderInWorldState(ws *WorldState, key OrderKey) {
	switch key.Button {
	case config.BT_Cab:
		if cabs, ok := ws.ConfirmedCabOrders[key.OwnerID]; ok {
			if cabs[key.Floor] {
				cabs[key.Floor] = false
			}
		}

	case config.BT_HallUp, config.BT_HallDown:
		if ws.ConfirmedHallOrders[key.Floor][key.Button] {
			ws.ConfirmedHallOrders[key.Floor][key.Button] = false
		}
	}
} */

/* 
func confirmOrderInWorldState(ws *WorldState, key OrderKey) bool {
	changed := false

	switch key.Button {
	case config.BT_Cab:
		if _, ok := ws.ConfirmedCabOrders[key.OwnerID]; !ok {
			ws.ConfirmedCabOrders[key.OwnerID] = make([]bool, config.N_FLOORS)
		}
		if !ws.ConfirmedCabOrders[key.OwnerID][key.Floor] {
			ws.ConfirmedCabOrders[key.OwnerID][key.Floor] = true
			changed = true
		}

	case config.BT_HallUp, config.BT_HallDown:
		if !ws.ConfirmedHallOrders[key.Floor][key.Button] {
			ws.ConfirmedHallOrders[key.Floor][key.Button] = true
			changed = true
		}
	}

	return changed
}


func clearOrderInWorldState(ws *WorldState, key OrderKey) bool {
	changed := false

	switch key.Button {
	case config.BT_Cab:
		if cabs, ok := ws.ConfirmedCabOrders[key.OwnerID]; ok {
			if cabs[key.Floor] {
				cabs[key.Floor] = false
				changed = true
			}
		}

	case config.BT_HallUp, config.BT_HallDown:
		if ws.ConfirmedHallOrders[key.Floor][key.Button] {
			ws.ConfirmedHallOrders[key.Floor][key.Button] = false
			changed = true
		}
	}

	return changed
}
 */


 func isOrderConfirmedInWorldState(ws *WorldState, key OrderKey) bool {
	switch key.Button {
	case config.BT_Cab:
		cabOrders, ok := ws.ConfirmedCabOrders[key.OwnerID]
		if !ok {
			return false
		}
		return cabOrders[key.Floor]

	case config.BT_HallUp, config.BT_HallDown:
		return ws.ConfirmedHallOrders[key.Floor][key.Button]
	}

	return false
}

func confirmOrderInWorldState(ws *WorldState, key OrderKey) {
	switch key.Button {
	case config.BT_Cab:
		if _, ok := ws.ConfirmedCabOrders[key.OwnerID]; !ok {
			ws.ConfirmedCabOrders[key.OwnerID] = make([]bool, config.N_FLOORS)
		}
		ws.ConfirmedCabOrders[key.OwnerID][key.Floor] = true

	case config.BT_HallUp, config.BT_HallDown:
		ws.ConfirmedHallOrders[key.Floor][key.Button] = true
	}
}

func clearOrderInWorldState(ws *WorldState, key OrderKey) {
	switch key.Button {
	case config.BT_Cab:
		if cabs, ok := ws.ConfirmedCabOrders[key.OwnerID]; ok {
			cabs[key.Floor] = false
		}

	case config.BT_HallUp, config.BT_HallDown:
		ws.ConfirmedHallOrders[key.Floor][key.Button] = false
	}
}