package ordermanagement

import (
	"fmt"
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
	OrderTxCh chan<- config.ButtonEvent, // send hall orders til andre heiser
	OrderRxCh <-chan config.ButtonEvent) {

	ws := NewWorldState(config.N_FLOORS)
	ot := make(OrderTracker)
	ws.Alive[myID] = true

	ws.States[myID] = config.ElevatorState{
		ID:          myID,
		Floor:       0, // Start at floor 0 instead of -1
		Behaviour:   config.BehIdle,
		Direction:   config.DirStop,
		CabRequests: make([]bool, config.N_FLOORS),
	}

	// Hardcoded peers for local testing (network broadcast doesn't work on localhost)
	for _, id := range []string{"elev1", "elev2", "elev3"} {
		if id != myID {
			ws.States[id] = config.ElevatorState{
				ID:          id,
				Floor:       0,
				Behaviour:   config.BehIdle,
				Direction:   config.DirStop,
				CabRequests: make([]bool, config.N_FLOORS),
			}
			ws.Alive[id] = true
		}
	}

	ordersOutCh <- buildOrders(&ws, myID)
	for {
		changed := false
		select {
		case btn := <-buttonCh:
			//apply button locoal 
			
			// add order to tracker for this elevator
			key := OrderKey{OwnerID: myID, Floor: btn.Floor, Button: btn.Button}
			ws.OrderTracker[key] = OrderInfo{
				SeenBy: map[string]bool{myID: true},
				Phase:  Confirmed,
			}

			key := OrderKey{OwnerID: myID, Floor: btn.Floor, Button: btn.Button}
        	ot[key] = OrderInfo{SeenBy: map[string]bool{myID: true}, Phase: CabConfirmed}

		   //add order to tracker for this elevator
			key := OrderKey{OwnerID: myID, Floor: btn.Floor, Button: btn.Button}
			ws.OrderTracker[key] = OrderInfo{
				SeenBy: map[string]bool{myID: true},
				Phase:  Confirmed,
			}
			// Broadcast orders to other elevators
			OrderTxCh <- btn

		case cl := <-clearCh:
			//clear button local
			// Broadcast clears to other elevators
			case OrderTxCh <- cl:

		case st := <-localStateCh:
			/* st.ID = myID
			if !statesEqual(ws.States[myID], st) {
				ws.States[myID] = st
				changed = true
			} */
			ws.States[st.ID] = st
			//changed = true

		case pst := <-peerStateCh:
			// Update peer state when received (overrides hardcoded defaults)
			ws.States[pst.ID] = pst
			//changed = true

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

		case peerHallBtn := <- OrderRxCh:
			// Receive hall orders from other elevators (including echo of own orders)

			//if gammel melding -> ignore (Juvan fikser <3)
			// 

			if peerHallBtn.Button == config.BT_HallUp || peerHallBtn.Button == config.BT_HallDown {
				if applyButton(&ws, myID, peerHallBtn) {
					changed = true
					fmt.Printf("[%s] Received hall order from peer: Floor %d, Button %d\n", myID, peerHallBtn.Floor, peerHallBtn.Button)
				}
			}

		}
		if changed {
			//fmt.Printf("buildOrders\n")
			orders := buildOrders(&ws, myID)
			ordersOutCh <- orders
		}
	}
}

// initialisere
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

// apply button til worldstate
func applyButton(ws *WorldState, myID string, btn config.ButtonEvent) bool {
	changed := false

	st := ws.States[myID]
	/* if st.ID == "" {
		st.ID = myID
		st.CabRequests = make([]bool, ws.NumFloors)
	} */

	switch btn.Button {
	case config.BT_Cab:
		if !st.CabRequests[btn.Floor] {
			st.CabRequests[btn.Floor] = true
			ws.States[myID] = st
			changed = true
		}
	case config.BT_HallUp, config.BT_HallDown:
		if ws.HallRequests[btn.Floor][btn.Button].Phase != HallConfirmed {
			ws.HallRequests[btn.Floor][btn.Button].Phase = HallConfirmed
			changed = true
		}
		// marker at jeg har sett den
		ws.HallRequests[btn.Floor][btn.Button].SeenBy[myID] = 1

	}
	return changed
}

// clear de som skal cleares
func applyClear(ws *WorldState, myID string, ce config.ClearEvent) bool {
	floor := ce.Floor
	changed := false
	if ce.ClearCab {
		st := ws.States[myID]
		if st.CabRequests[floor] {
			st.CabRequests[floor] = false
			ws.States[myID] = st
			changed = true
		}
	}

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
	// MIDLERTIDIG
	o := NewOrders(config.N_FLOORS)

	state := ws.States[myID]

	for floor := 0; floor < config.N_FLOORS; floor++ {
		o.Cab[floor] = state.CabRequests[floor]
	}

	// hall: alle confirmed (midlertidig uten assigner)
	for floor := 0; floor < config.N_FLOORS; floor++ {
		o.Hall[floor][0] = (ws.HallRequests[floor][0].Phase == HallConfirmed)
		o.Hall[floor][1] = (ws.HallRequests[floor][1].Phase == HallConfirmed)
	}
	return o
}

// bygger ordere objektet som skal sendes til FSM
func buildOrders(ws *WorldState, myID string) Orders {
	inputAssigner := buildAssignerInput(ws)
	path := "./hall_request_assigner/hall_request_assigner"

	assignments, err := CallAssigner(path, inputAssigner)
	if err != nil {
		fmt.Printf("Assigner error: %v\n", err)
		return buildOrdersAllHall(ws, myID) //hvis den feiler, får alle heisene dens ordere
	}

	assignedHall, ok := assignments[myID]
	fmt.Printf("%v\n", assignedHall)
	if !ok {
		fmt.Printf("MyID %s not in assigner output\n", myID)
		return buildOrdersAllHall(ws, myID)
	}

	o := NewOrders(config.N_FLOORS)
	state := ws.States[myID]
	for floor := 0; floor < config.N_FLOORS; floor++ {
		o.Cab[floor] = state.CabRequests[floor]
		o.Hall[floor][0] = assignedHall[floor][0]
		o.Hall[floor][1] = assignedHall[floor][1]
	}

	return o
}

// konvererer input til assigner
func buildAssignerInput(ws *WorldState) AssignerInput {
	hallRequests := make([][]bool, config.N_FLOORS)
	for floor := 0; floor < config.N_FLOORS; floor++ {
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

// initialiserer world state funksjonen
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
