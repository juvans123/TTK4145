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
}