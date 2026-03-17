package ordermanagement

import (
	"fmt"
	"heis/config"
	"time"
)

func Run(
	myID string,
	buttonCh <-chan config.ButtonEvent,
	clearCh <-chan config.ClearEvent,
	localStateCh <-chan config.ElevatorState,
	peerStateCh <-chan config.ElevatorState,
	peerEventCh <-chan config.PeerEvent,
	ordersOutCh chan<- Orders,
	OrderOutCh chan<- OrderMsg, // OM -> Network
	OrderInCh <-chan OrderMsg, // OM <- Network
	setButtonLight chan<- config.LightState,
) {
	worldState := NewWorldState()
	localOrderView := make(OrderTracker)

	orderBroadcast := time.NewTicker(100 * time.Millisecond)
	defer orderBroadcast.Stop()

	worldState.Alive[myID] = true
	worldState.States[myID] = config.ElevatorState{
		ID:          myID,
		Floor:       0,
		Behaviour:   config.BehIdle,
		Direction:   config.DirStop,
		CabRequests: make([]bool, config.N_FLOORS),
	}

	ordersOutCh <- buildMyLocalOrders(&worldState, myID)

mainLoop:
	for {
		changed := false

		select {
		case btn := <-buttonCh:

			ownerID := ownerForButton(myID, btn.Button)
			key := makeOrderKey(ownerID, btn.Floor, btn.Button)
			localOrder := localOrderView[key]

			if localOrder.Phase == Unconfirmed || localOrder.Phase == Confirmed {
				continue mainLoop
			}

			localOrder = updateLocalOrderPhase(localOrderView, key, Unconfirmed, myID)

			if allAliveHaveSeen(localOrder.SeenBy, worldState.Alive) {
				if confirmOrderInWorldState(&worldState, key) {
					changed = true
				}
				
				localOrder = updateLocalOrderPhase(localOrderView, key, Confirmed, myID)
			}

		case cl := <-clearCh:
			clears := []struct {
				shouldClear bool
				button      config.ButtonType
			}{
				{cl.ClearCab, config.BT_Cab},
				{cl.ClearHallUp, config.BT_HallUp},
				{cl.ClearHallDown, config.BT_HallDown},
			}

			for _, clearInfo := range clears {
				if !clearInfo.shouldClear {
					continue
				}

				ownerID := ownerForButton(myID, clearInfo.button)
				key := makeOrderKey(ownerID, cl.Floor, clearInfo.button)
				localOrder := localOrderView[key]

				if localOrder.Phase != Confirmed {
					continue
				}

				localOrder = updateLocalOrderPhase(localOrderView, key, Served, myID)

				if clearOrderInWorldState(&worldState, key) {
					changed = true
				}
				changed = true

				if allAliveHaveSeen(localOrder.SeenBy, worldState.Alive) {
					delete(localOrderView, key)
				}

			}

		case newLocalState := <-localStateCh:
			prevLocalState := worldState.States[myID]
			newLocalState.ID = myID  //fjerne denne linjer og skrive myID direkte i linjen under
			worldState.States[newLocalState.ID] = newLocalState

			if prevLocalState.Immobile != newLocalState.Immobile {
				changed = true
			}

		case newPeerState := <-peerStateCh:
			prevPeerState := worldState.States[newPeerState.ID]
			worldState.States[newPeerState.ID] = newPeerState

			if prevPeerState.Immobile != newPeerState.Immobile {
				changed = true
			}

		case peer := <-peerEventCh:
			prev, exists := worldState.Alive[peer.PeerID]
			if !exists || prev != peer.Alive {
				worldState.Alive[peer.PeerID] = peer.Alive
				changed = true
			}
			

			if peer.Alive {
				for key, info := range localOrderView {
					if key.OwnerID == myID && key.Button == config.BT_Cab && info.Phase == Served {
						delete(localOrderView, key)
					}
				}
			}

		//her vil jeg kalle kanalen peerAlivenesschange, inni peerevent så skal jeg fjerne peer før id, og døpe om peerevent til peeralivness

		case peerOrder := <-OrderInCh:
			key := makeOrderKey(peerOrder.OwnerID, peerOrder.Floor, peerOrder.Button)
			localOrder := localOrderView[key]

			if localOrder.SeenBy == nil {
				localOrder.SeenBy = make(map[string]bool)
				localOrder.Phase = NoOrder
			}

			if peerOrder.Phase > localOrder.Phase {
				localOrder.Phase = peerOrder.Phase
				localOrder.SeenBy = make(map[string]bool)
			} else if peerOrder.Phase < localOrder.Phase {
				continue mainLoop
			}

			for id, seen := range peerOrder.SeenBy {
				if seen {
					localOrder.SeenBy[id] = true
				}
			}

			localOrder.SeenBy[myID] = true
			localOrderView[key] = localOrder //heller skriv localOrderView[key].seenBy[myID] = true, og fjerne linjen over

			if localOrder.Phase == Unconfirmed && !allAliveHaveSeen(localOrder.SeenBy, worldState.Alive) {
				continue mainLoop
			}


			switch localOrder.Phase {
			case Unconfirmed:
				if confirmOrderInWorldState(&worldState, key) {
					changed = true
				}
				localOrder = updateLocalOrderPhase(localOrderView, key, Confirmed, myID)
				changed = true

				//lage update fase funksjoner

			case Confirmed:

				if confirmOrderInWorldState(&worldState, key) {
					changed = true
				}

			case Served:
				if clearOrderInWorldState(&worldState, key) {
					changed = true
				}
				delete(localOrderView, key)
				changed = true
			}



		case <-orderBroadcast.C:
			for key, localOrder := range localOrderView {
				if localOrder.Phase == NoOrder {
					continue
				}
				OrderOutCh <- OrderMsg{
					OwnerID: key.OwnerID,
					Floor:   key.Floor,
					Button:  key.Button,
					Phase:   localOrder.Phase,
					SeenBy:  copySeenBy(localOrder.SeenBy),
				}
			}
		}

		if changed {
			setButtonLight <- buildLightState(&worldState, myID)
			ordersOutCh <- buildMyLocalOrders(&worldState, myID)
		}
	}
}

//-------


func updateLocalOrderPhase(view OrderTracker, key OrderKey,newPhase OrderPhase, myID string,) OrderInfo{
	localOrder := view[key]
	localOrder.Phase = newPhase
	localOrder.SeenBy = map[string]bool{myID: true}
	view[key] = localOrder
	return localOrder
}

func ownerForButton(myID string, button config.ButtonType) string {
	if button == config.BT_Cab {
		return myID
	}
	return ""
}

func makeOrderKey(ownerID string, floor int, button config.ButtonType) OrderKey {
	return OrderKey{
		OwnerID: ownerID,
		Floor:   floor,
		Button:  button,
	}
}

func copySeenBy(src map[string]bool) map[string]bool {
	dst := make(map[string]bool)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func allAliveHaveSeen(seenBy map[string]bool, alive map[string]bool) bool {
	for id, isAlive := range alive {
		if !isAlive {
			continue
		}
		if !seenBy[id] {
			return false
		}
	}
	return true
}

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

func NewOrders(numFloors int) Orders {
	orders := Orders{
		Cab:  make([]bool, numFloors),
		Hall: make([][]bool, numFloors),
	}

	for floor := 0; floor < numFloors; floor++ {
		orders.Hall[floor] = make([]bool, 2)
	}

	return orders
}

func buildMyLocalOrders(ws *WorldState, myID string) Orders {
	inputAssigner := buildAssignerInput(myID, ws)
	path := "./hall_request_assigner/hall_request_assigner"

	assignments, err := CallAssigner(path, inputAssigner)
	if err != nil {
		fmt.Printf("Assigner error: %v\n", err)
		return buildCabOnlyOrders(ws, myID)
	}

	myAssignedHall, ok := assignments[myID]
	if !ok {
		return buildCabOnlyOrders(ws, myID)
	}

	myLocalOrders := NewOrders(config.N_FLOORS)

	confirmedCab, ok := ws.ConfirmedCabOrders[myID]
	if ok && len(confirmedCab) == config.N_FLOORS {
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
		return cabOnlyOrders
	}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		cabOnlyOrders.Cab[floor] = confirmedCab[floor]
	}

	return cabOnlyOrders
}

func buildAssignerInput(myID string, ws *WorldState) AssignerInput {
	hallRequests := make([][]bool, config.N_FLOORS)
	for floor := 0; floor < config.N_FLOORS; floor++ {
		hallRequests[floor] = make([]bool, 2)
		hallRequests[floor][config.BT_HallUp] = ws.ConfirmedHallOrders[floor][config.BT_HallUp]
		hallRequests[floor][config.BT_HallDown] = ws.ConfirmedHallOrders[floor][config.BT_HallDown]
	}

	states := make(map[string]config.ElevatorState)
	for id, state := range ws.States {
		if !ws.Alive[id] {
			continue
		}

		if state.Immobile && id != myID {
			continue
		}

		confirmedCab, ok := ws.ConfirmedCabOrders[id]
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

func buildLightState(ws *WorldState, myID string) config.LightState {
	ls := config.LightState{
		Cab: make([]bool, config.N_FLOORS),
	}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		ls.Hall[floor][config.BT_HallUp] = ws.ConfirmedHallOrders[floor][config.BT_HallUp]
		ls.Hall[floor][config.BT_HallDown] = ws.ConfirmedHallOrders[floor][config.BT_HallDown]
	}

	if cabOrders, ok := ws.ConfirmedCabOrders[myID]; ok && len(cabOrders) == config.N_FLOORS {
		copy(ls.Cab, cabOrders)
	}

	return ls
}

func HasOrders(orders *Orders) bool {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		if HasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
}
