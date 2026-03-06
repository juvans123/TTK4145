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
	OrderTxCh chan<- OrderMsg, // send hall orders til andre heiser
	OrderRxCh <-chan OrderMsg) {

	ws := NewWorldState()
	localOrderView := make(OrderTracker)
	ws.Alive[myID] = true
	peerID := ""
	

	if myID == "elev1"{
		peerID = "elev2"
	} else{
		peerID = "elev1"
	}

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

	ordersOutCh <- buildMyLocalOrders(&ws, myID)
	for {
		changed := false
		select {
		case btn := <-buttonCh:
			//apply button locoal

			// add order to tracker for this elevator
			key := OrderKey{OwnerID: myID, Floor: btn.Floor, Button: btn.Button}
			localOrderView[key] = OrderInfo{SeenBy: map[string]bool{myID: true}, Phase: Confirmed}

			//lage message for sending
			msg := OrderMsg{
				OwnerID: myID,
				Floor:   btn.Floor,
				Button:  btn.Button,
				SeenBy:  map[string]bool{myID: true},
				Phase:   Confirmed,
			}

			OrderTxCh <- msg

		case cl := <-clearCh:
			// Definer hvilke clear-typer som skal håndteres
			clears := []struct {
				shouldClear bool
				button      config.ButtonType
			}{
				{cl.ClearCab, config.BT_Cab},
				{cl.ClearHallUp, config.BT_HallUp},
				{cl.ClearHallDown, config.BT_HallDown},
			}

			// For hver clear-type som er true, oppdater tracker og send melding
			for _, clearInfo := range clears {
				if clearInfo.shouldClear {
					key := OrderKey{OwnerID: myID, Floor: cl.Floor, Button: clearInfo.button}
					localOrderView[key] = OrderInfo{SeenBy: map[string]bool{myID: true}, Phase: NoOrder}

					// Lage melding for sending
					msg := OrderMsg{
						OwnerID: myID,
						Floor:   cl.Floor,
						Button:  clearInfo.button,
						SeenBy:  map[string]bool{myID: true},
						Phase:   NoOrder,
					}

					OrderTxCh <- msg
				}
			}

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

		case peerOrder := <-OrderRxCh:


			if peerOrder.Phase == Confirmed {
				//setter til seen i egen
				key := OrderKey{OwnerID: myID, Floor: peerOrder.Floor, Button: peerOrder.Button}
				localOrderView[key] = OrderInfo{SeenBy: map[string]bool{myID: true, peerID: true}, Phase: Confirmed}
				// Hvis begge har sett orderen Sett ordren til confirmed i world state

				if localOrderView[key].SeenBy[myID] && localOrderView[key].SeenBy[peerID]{
					switch peerOrder.Button {
					case config.BT_Cab:
						// make sure we've allocated the slice for this owner
						if _, ok := ws.ConfirmedCabOrders[peerOrder.OwnerID]; !ok {
							ws.ConfirmedCabOrders[peerOrder.OwnerID] = make([]bool, config.N_FLOORS)
						}
						ws.ConfirmedCabOrders[peerOrder.OwnerID][peerOrder.Floor] = true
						changed = true
					case config.BT_HallUp, config.BT_HallDown:
						ws.ConfirmedHallOrders[peerOrder.Floor][peerOrder.Button] = true
						changed = true
					}
					
				}
			}
			if peerOrder.Phase == NoOrder {
				
				key := OrderKey{OwnerID: myID, Floor: peerOrder.Floor, Button: peerOrder.Button}
				localOrderView[key] = OrderInfo{SeenBy: map[string]bool{myID: false, peerID: false}, Phase: NoOrder}
				// Hvis begge har sett orderen Sett ordren til confirmed i world state

				if !localOrderView[key].SeenBy[myID] && !localOrderView[key].SeenBy[peerID]{
				// Sett ordren til none i world state
		
					switch peerOrder.Button {
					case config.BT_Cab:
						if _, ok := ws.ConfirmedCabOrders[peerOrder.OwnerID]; ok {
							ws.ConfirmedCabOrders[peerOrder.OwnerID][peerOrder.Floor] = false
						}
						changed = true
					case config.BT_HallUp, config.BT_HallDown:
						ws.ConfirmedHallOrders[peerOrder.Floor][peerOrder.Button] = false
						changed = true
					}
				}
			}

		}
		if changed {
			//fmt.Printf("buildOrders\n")
			orders := buildMyLocalOrders(&ws, myID)
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
/* func applyButton(ws *WorldState, myID string, btn config.ButtonEvent) bool {
	changed := false

	st := ws.States[myID]
	/* if st.ID == "" {
		st.ID = myID
		st.CabRequests = make([]bool, ws.NumFloors)
	} 

	switch btn.Button {
	case config.BT_Cab:
		if _, ok := ws.ConfirmedCabOrders[myID]; !ok {
			ws.ConfirmedCabOrders[myID] = make([]bool, config.N_FLOORS)
		}
		if !ws.ConfirmedCabOrders[myID][btn.Floor] {
			ws.ConfirmedCabOrders[myID][btn.Floor] = true
			changed = true
		}
	case config.BT_HallUp, config.BT_HallDown:
		if !ws.ConfirmedHallOrders[btn.Floor][btn.Button] {
			ws.ConfirmedHallOrders[btn.Floor][btn.Button] = true
			changed = true
		}
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

	if ce.ClearHallUp && ws.ConfirmedHallOrders[floor][config.BT_HallUp] {
		ws.ConfirmedHallOrders[floor][config.BT_HallUp] = false
		changed = true
	}
	if ce.ClearHallDown && ws.ConfirmedHallOrders[floor][config.BT_HallDown] {
		ws.ConfirmedHallOrders[floor][config.BT_HallDown] = false
		changed = true
	}

	return changed
}
/* 
func buildOrdersAllHall(ws *WorldState, myID string) Orders {
	// MIDLERTIDIG
	o := NewOrders(config.N_FLOORS)

	state := ws.States[myID]

	for floor := 0; floor < config.N_FLOORS; floor++ {
		o.Cab[floor] = state.CabRequests[floor]
	}

	// hall: alle confirmed (midlertidig uten assigner)
	for floor := 0; floor < config.N_FLOORS; floor++ {
		o.Hall[floor][0] = (ws.HallRequests[floor][0].Phase == Confirmed)
		o.Hall[floor][1] = (ws.HallRequests[floor][1].Phase == Confirmed)
	}
	return o
}
 */
func buildMyLocalOrders(ws *WorldState, myID string) Orders {
	inputAssigner := buildAssignerInput(ws)
	path := "./hall_request_assigner/hall_request_assigner"

	assignments, err := CallAssigner(path, inputAssigner)
	if err != nil {
		fmt.Printf("Assigner error: %v\n", err)
		return buildCabOnlyOrders(ws, myID)
	}

	myAssignedHall, ok := assignments[myID]
	fmt.Printf("Assigned hall for %s: %+v\n", myID, myAssignedHall)
	if !ok {
		fmt.Printf("MyID %s not in assigner output\n", myID)
		return buildCabOnlyOrders(ws, myID)
	}

	myLocalOrders := NewOrders(config.N_FLOORS)

	confirmedCab, ok := ws.ConfirmedCabOrders[myID]
	if ok && len(confirmedCab) == config.N_FLOORS{
		for floor := 0; floor < config.N_FLOORS; floor++ {
			myLocalOrders.Cab[floor] = confirmedCab[floor]
		}
	}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		myLocalOrders.Hall[floor][config.BT_HallUp] = myAssignedHall[floor][config.BT_HallUp]
		myLocalOrders.Hall[floor][config.BT_HallDown] = myAssignedHall[floor][config.BT_HallDown]
	}

	return myLocalOrders
}
 
func buildCabOnlyOrders(ws *WorldState, myID string) Orders {
	cabOnlyOrders := NewOrders(config.N_FLOORS)

	confirmedCab, ok := ws.ConfirmedCabOrders[myID]
	if !ok {
		return cabOnlyOrders
	}

	if len(confirmedCab) != config.N_FLOORS {
		fmt.Printf("Warning: ConfirmedCabOrders for %s has wrong length\n", myID)
		return cabOnlyOrders
	}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		cabOnlyOrders.Cab[floor] = confirmedCab[floor]
	}

	return cabOnlyOrders
}


func buildAssignerInput(ws *WorldState) AssignerInput {
	hallRequests := make([][]bool, config.N_FLOORS)
	for floor := 0; floor < config.N_FLOORS; floor++ {
		hallRequests[floor] = make([]bool, 2)
		hallRequests[floor][config.BT_HallUp] = ws.ConfirmedHallOrders[floor][config.BT_HallUp]
		hallRequests[floor][config.BT_HallDown] = ws.ConfirmedHallOrders[floor][config.BT_HallDown]
	}

	states := make(map[string]config.ElevatorState)
	for id, state := range ws.States {
		alive, ok := ws.Alive[id];
		if ok && !alive {
			continue
		}

		confirmedCab, ok := ws.ConfirmedCabOrders[id];
		if ok && len(confirmedCab) == config.N_FLOORS {
			cabCopy := make([]bool, config.N_FLOORS)
			copy(cabCopy, confirmedCab)
			state.CabRequests = cabCopy
		}

		states[id] = state
	}

	return AssignerInput{
		HallRequests: hallRequests,
		States:       states,
	}
}

func NewWorldState() WorldState {
	return WorldState{
		ConfirmedHallOrders: [config.N_FLOORS][2]bool{},
		ConfirmedCabOrders:  make(map[string][]bool),
		States:              make(map[string]config.ElevatorState),
		Alive:               make(map[string]bool),
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
		orders.Hall[floor][config.BT_HallUp] ||
		orders.Hall[floor][config.BT_HallDown]
}
