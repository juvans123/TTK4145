package ordermanagement

import (
	"heis/config"
)

//const NumFloors = 4

func Run(
	myID string,
	buttonCh <-chan config.ButtonEvent, // lokale knappetrykk
	clearCh <-chan config.ClearEvent, // fra FSM når dør ordre er servert
	localStateCh <-chan config.ElevatorState, // public state fra lokal FSM
	peerStateCh <-chan config.ElevatorState, // states fra andre heiser (via nettverk)
	peerUpdateCh <-chan config.PeerUpdate, // (id, alive/dead) fra supervisor
	ordersOutCh chan<- Orders, // snapshot til FSM

) {
	numFloors := 4
	ws := NewWorldState(numFloors)
	ws.Alive[myID] = true

	ws.States[myID] = config.ElevatorState{
		ID:          myID,
		Floor:       -1,
		CabRequests: make([]bool, numFloors),
	}

	ordersOutCh <- buildOrders(&ws, myID)
	for {
		changed := false
		select {
		case btn := <-buttonCh:
			if applyButton(&ws, myID, btn) {
				changed = true
			}
		case cl := <-clearCh:
			if applyClear(&ws, myID, cl) {
				changed = true
			}

		case st := <-localStateCh:
			ws.States[st.ID] = st
			changed = true

		case pst := <-peerStateCh:
			ws.States[pst.ID] = pst
			changed = true

		case pu := <-peerUpdateCh:
			currentAliveStatus, exists := ws.Alive[pu.ID]
			if !exists {
				// Vi har aldri sett denne heisen før
				ws.Alive[pu.ID] = pu.Alive
				changed = true
			} else if currentAliveStatus != pu.Alive {
				// Status har endret seg (alive -> dead eller motsatt)
				ws.Alive[pu.ID] = pu.Alive
				changed = true
			}
		}
		if changed {
			orders := buildOrders(&ws, myID)
			ordersOutCh <- orders
		}
	}
}

func NewOrders(numFloors int) Orders {
	o := Orders{
		Cab:  make([]bool, numFloors),
		Hall: make([][]bool, numFloors),
	}

	for floor := 0; floor < numFloors; floor++ {
		o.Hall[floor] = make([]bool, 2)
	}

	return o
}

func applyButton(ws *WorldState, myID string, btn config.ButtonEvent) bool {
	changed := false

	st := ws.States[myID]
	if st.ID == "" {
		st.ID = myID
		st.CabRequests = make([]bool, ws.NumFloors)
	}

	switch btn.Button {
	case config.BT_Cab:
		if !st.CabRequests[btn.Floor] {
			st.CabRequests[btn.Floor] = true
			ws.States[myID] = st
			changed = true
		}
	case config.BT_HallUp, config.BT_HallDown:
		//if-sjekk?
		if ws.HallRequests[btn.Floor][btn.Button].Phase != HallConfirmed {
			ws.HallRequests[btn.Floor][btn.Button].Phase = HallConfirmed
			changed = true
		}
		// marker at jeg har sett den
		ws.HallRequests[btn.Floor][btn.Button].SeenBy[myID] = 1

	}
	return changed
}

func applyClear(ws *WorldState, myID string, ce config.ClearEvent) bool {
	changed := false
	if ce.ClearCab {
		st := ws.States[myID]
		if st.CabRequests[ce.Floor] {
			st.CabRequests[ce.Floor] = false
			ws.States[myID] = st
			changed = true
		}
	}

	floor := ce.Floor

	if ce.ClearHallUp && ws.HallRequests[floor][config.BT_HallUp].Phase != HallNone {
		ws.HallRequests[floor][config.BT_HallUp].Phase = HallNone
		ws.HallRequests[floor][config.BT_HallUp].SeenBy = make(map[string]uint8)
		changed = true
	}
	if ce.ClearHallDown && ws.HallRequests[floor][config.BT_HallDown].Phase != HallNone {
		ws.HallRequests[floor][config.BT_HallDown].Phase = HallNone
		ws.HallRequests[floor][config.BT_HallDown].SeenBy = make(map[string]uint8)
		changed = true
	}

	return changed
}

func buildOrdersAllHall(ws *WorldState, myID string) Orders {
	o := NewOrders(ws.NumFloors)

	state := ws.States[myID]

	for floor := 0; floor < ws.NumFloors; floor++ {
		o.Cab[floor] = state.CabRequests[floor]
	}

	// hall: alle confirmed (midlertidig uten assigner)
	for floor := 0; floor < ws.NumFloors; floor++ {
		o.Hall[floor][0] = (ws.HallRequests[floor][0].Phase == HallConfirmed)
		o.Hall[floor][1] = (ws.HallRequests[floor][1].Phase == HallConfirmed)
	}
	return o
}

func buildOrders(ws *WorldState, myID string) Orders {
	inputAssigner := buildAssignerInput(ws)
	path := "./hall_request_assigner/hall_request_assigner"
	assignments, err := CallAssigner(path, inputAssigner)
	if err != nil {
		return buildOrdersAllHall(ws, myID)
	}

	assignedHall, ok := assignments[myID]
	if !ok {
		return buildOrdersAllHall(ws, myID)
	}

	o := NewOrders(ws.NumFloors)
	state := ws.States[myID]
	for floor := 0; floor < ws.NumFloors; floor++ {
		o.Cab[floor] = state.CabRequests[floor]
		o.Hall[floor][0] = assignedHall[floor][0]
		o.Hall[floor][1] = assignedHall[floor][1]
	}

	return o
}

func buildAssignerInput(ws *WorldState) AssignerInput {
	hallRequests := make([][]bool, ws.NumFloors)
	for floor := 0; floor < ws.NumFloors; floor++ {
		hallRequests[floor] = make([]bool, 2)
		hallRequests[floor][0] = (ws.HallRequests[floor][0].Phase == HallConfirmed)
		hallRequests[floor][1] = (ws.HallRequests[floor][1].Phase == HallConfirmed)
	}

	states := make(map[string]config.ElevatorState)
	for id, st := range ws.States {
		if alive, ok := ws.Alive[id]; ok && !alive {
			continue
		}
		states[id] = st
	}

	return AssignerInput{HallRequests: hallRequests, States: states}
}

func NewWorldState(numFloors int) WorldState {
	ws := WorldState{
		NumFloors:    numFloors,
		HallRequests: make([][]HallOrderState, numFloors),
		States:       make(map[string]config.ElevatorState),
		Alive:        make(map[string]bool),
	}

	for floor := 0; floor < numFloors; floor++ {
		ws.HallRequests[floor] = make([]HallOrderState, 2) // 0=up,1=down
		for dir := 0; dir < 2; dir++ {
			ws.HallRequests[floor][dir] = HallOrderState{
				Phase:  HallNone,
				SeenBy: make(map[string]uint8),
			}
		}
	}
	return ws
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
		orders.Hall[floor][config.BT_HallUp] ||
		orders.Hall[floor][config.BT_HallDown]
}

/* func addOrderFromButtonEvent(btn config.ButtonEvent, orders *Orders) {
	switch btn.Button {
	case config.BT_Cab:
		orders.Cab[btn.Floor] = true

	case config.BT_HallUp:
		orders.Hall[btn.Floor][config.BT_HallUp] = true

	case config.BT_HallDown:
		orders.Hall[btn.Floor][config.BT_HallDown] = true
	}
} */

/*
func ClearAtFloor(orders *Orders, floor int, travelDir config.TravelDirection) {
	if floor >= 0 && floor < len(orders.Cab) {
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

} */
