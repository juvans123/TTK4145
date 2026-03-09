package ordermanagement

import (
	"fmt"
	"heis/config"
)

func Run(
	myID string,
	buttonCh <-chan config.ButtonEvent,
	clearCh <-chan config.ClearEvent,
	localStateCh <-chan config.ElevatorState,
	peerStateCh <-chan config.ElevatorState,
	peerEventCh <-chan config.PeerEvent,
	ordersOutCh chan<- Orders,
	OrderTxCh chan<- OrderMsg,
	OrderRxCh <-chan OrderMsg,
	setButtonLight chan<- config.LightState,
) {
	ws := NewWorldState()
	localOrderView := make(OrderTracker)

	ws.Alive[myID] = true
	ws.States[myID] = config.ElevatorState{
		ID:          myID,
		Floor:       0,
		Behaviour:   config.BehIdle,
		Direction:   config.DirStop,
		CabRequests: make([]bool, config.N_FLOORS),
	}

	// Kun states for lokal testing, ikke alive
	for _, id := range []string{"elev1", "elev2", "elev3"} {
		if id != myID {
			ws.States[id] = config.ElevatorState{
				ID:          id,
				Floor:       0,
				Behaviour:   config.BehIdle,
				Direction:   config.DirStop,
				CabRequests: make([]bool, config.N_FLOORS),
			}
		}
	}

	ordersOutCh <- buildMyLocalOrders(&ws, myID)

mainLoop:
	for {
		changed := false

		select {
		case btn := <-buttonCh:
			ownerID := ownerForButton(myID, btn.Button)
			key := makeOrderKey(ownerID, btn.Floor, btn.Button)

			info := localOrderView[key] // zero-value hvis key ikke finnes

			if info.Phase != NoOrder {
				continue mainLoop
			}

			info.Phase = Unconfirmed
			info.SeenBy = map[string]bool{myID: true}
			localOrderView[key] = info

			OrderTxCh <- OrderMsg{
				OwnerID: ownerID,
				Floor:   btn.Floor,
				Button:  btn.Button,
				Phase:   Unconfirmed,
				SeenBy:  copySeenBy(info.SeenBy),
			}

			// Hvis jeg er eneste alive, kan ordren bekreftes med en gang
			fmt.Printf("Her er jeg:  %s", myID)
			if allAliveHaveSeen(info.SeenBy, ws.Alive) {
				fmt.Printf("Id %s is the only one alive", myID)
				if confirmOrderInWorldState(&ws, key) {
					changed = true
				}
				info.Phase = Confirmed
				info.SeenBy = make(map[string]bool)
				localOrderView[key] = info
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

				info := localOrderView[key]

				// Start bare clear hvis ordren faktisk er confirmed lokalt
				if info.Phase != Confirmed {
					continue
				}

				info.Phase = Served
				info.SeenBy = map[string]bool{myID: true}
				localOrderView[key] = info

				OrderTxCh <- OrderMsg{
					OwnerID: ownerID,
					Floor:   cl.Floor,
					Button:  clearInfo.button,
					Phase:   Served,
					SeenBy:  copySeenBy(info.SeenBy),
				}

				// Hvis jeg er eneste alive, kan clear bekreftes med en gang
				if allAliveHaveSeen(info.SeenBy, ws.Alive) {
					if clearOrderInWorldState(&ws, key) {
						changed = true
					}
					localOrderView[key] = OrderInfo{
						Phase:  NoOrder,
						SeenBy: make(map[string]bool),
					}
				}
			}

		case st := <-localStateCh:
			ws.States[st.ID] = st

		case pst := <-peerStateCh:
			ws.States[pst.ID] = pst

		case pe := <-peerEventCh:
			prev, exists := ws.Alive[pe.PeerID]
			if !exists || prev != pe.Alive {
				ws.Alive[pe.PeerID] = pe.Alive
				changed = true
				
				// Hvis en heis dør, rebroadcast alle ventende ordrer slik at de re-evalueres
				if !pe.Alive {
					for key, info := range localOrderView {
						if info.Phase == Unconfirmed || info.Phase == Served {
							OrderTxCh <- OrderMsg{
								OwnerID: key.OwnerID,
								Floor:   key.Floor,
								Button:  key.Button,
								Phase:   info.Phase,
								SeenBy:  copySeenBy(info.SeenBy),
							}
						}
					}
				}
			}

		case peerOrder := <-OrderRxCh:
			key := makeOrderKey(peerOrder.OwnerID, peerOrder.Floor, peerOrder.Button)

			info := localOrderView[key]
			if info.SeenBy == nil {
				info.SeenBy = make(map[string]bool)
				info.Phase = NoOrder
			}

			shouldRebroadcast := false

			/* 	// Ny fase: start ny seenBy-runde
			if info.Phase != peerOrder.Phase {
				info.Phase = peerOrder.Phase
				info.SeenBy = make(map[string]bool)
				shouldRebroadcast = true
			} */
			if peerOrder.Phase == Served && !isConfirmedInWorldState(&ws, key) {
				continue mainLoop
			}

			if peerOrder.Phase > info.Phase {
				// Peer har en nyere fase enn meg -> oppgrader
				info.Phase = peerOrder.Phase
				info.SeenBy = make(map[string]bool)
				shouldRebroadcast = true
			} else if peerOrder.Phase < info.Phase {
				// Peer har en eldre fase enn meg -> ignorer meldingen
				localOrderView[key] = info
				continue mainLoop
			}

			// Merge seenBy fra meldingen
			for id, seen := range peerOrder.SeenBy {
				if seen && !info.SeenBy[id] {
					info.SeenBy[id] = true
					shouldRebroadcast = true
				}
			}

			// Marker at jeg også har sett meldingen
			if !info.SeenBy[myID] {
				info.SeenBy[myID] = true
				shouldRebroadcast = true
			}

			localOrderView[key] = info

			if shouldRebroadcast {
				OrderTxCh <- OrderMsg{
					OwnerID: key.OwnerID,
					Floor:   key.Floor,
					Button:  key.Button,
					Phase:   info.Phase,
					SeenBy:  copySeenBy(info.SeenBy),
				}
			}

			if !allAliveHaveSeen(info.SeenBy, ws.Alive) {
				continue mainLoop
			}

			switch info.Phase {
			case Unconfirmed:
				if confirmOrderInWorldState(&ws, key) {
					changed = true
				}
				info.Phase = Confirmed
				info.SeenBy = make(map[string]bool)
				localOrderView[key] = info
				//fmt.Printf("[OM %s] CONFIRMED %+v\n", myID, key)

			case Served:
				if clearOrderInWorldState(&ws, key) {
					changed = true
				}
				localOrderView[key] = OrderInfo{
					Phase:  NoOrder,
					SeenBy: make(map[string]bool),
				}
				//fmt.Printf("[OM %s] CLEARED %+v\n", myID, key)
			}
		}

		if changed {
			setButtonLight <- buildLightState(&ws, myID)
			orders := buildMyLocalOrders(&ws, myID)
			//fmt.Printf("[OM %s] sender ny order til FSM: %+v\n", myID, orders)
			ordersOutCh <- orders
		}
	}
}

//-------

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

func isConfirmedInWorldState(ws *WorldState, key OrderKey) bool {
	switch key.Button {
	case config.BT_Cab:
		if cabs, ok := ws.ConfirmedCabOrders[key.OwnerID]; ok {
			return cabs[key.Floor]
		}
		return false
	case config.BT_HallUp, config.BT_HallDown:
		return ws.ConfirmedHallOrders[key.Floor][key.Button]
	}
	return false
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

// initialisere
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
	inputAssigner := buildAssignerInput(ws)
	//fmt.Printf("InputAssigner %+v\n", inputAssigner)
	path := "./hall_request_assigner/hall_request_assigner"

	assignments, err := CallAssigner(path, inputAssigner)
	if err != nil {
		fmt.Printf("Assigner error: %v\n", err)
		return buildCabOnlyOrders(ws, myID)
	}

	myAssignedHall, ok := assignments[myID]
	//fmt.Printf("Assigned hall for %s: %+v\n", myID, myAssignedHall)
	if !ok {
		fmt.Printf("MyID %s not in assigner output\n", myID)
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
		alive, ok := ws.Alive[id]
		if ok && !alive {
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

/* func allAliveHaveSeen(seenBy map[string]bool, alive map[string]bool) bool {
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

func copySeenBy(src map[string]bool) map[string]bool {
	dst := make(map[string]bool)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func initializeLocalOrderTracker(localOrderView OrderTracker, myID string) {
	buttons := []config.ButtonType{config.BT_HallUp, config.BT_HallDown, config.BT_Cab}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for _, button := range buttons {
			localOrderView[OrderKey{
				OwnerID: myID,
				Floor:   floor,
				Button:  button,
			}] = OrderInfo{
				Phase:  NoOrder,
				SeenBy: map[string]bool{myID: true},
			}
		}
	}
} */

func HasOrders(orders *Orders) bool {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		if HasOrderAtFloor(orders, floor) {
			return true
		}
	}
	return false
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

/* if peerOrder.Phase == Confirmed {
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
} *
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
}*/
