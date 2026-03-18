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
	ws := NewWorldState()
	localOrderView := make(OrderTracker)

	SendTicker := time.NewTicker(100 * time.Millisecond)
	defer SendTicker.Stop()

	ws.Alive[myID] = true
	ws.States[myID] = config.ElevatorState{
		ID:          myID,
		Floor:       0,
		Behaviour:   config.BehIdle,
		Direction:   config.DirStop,
		CabRequests: make([]bool, config.N_FLOORS),
	}

	ordersOutCh <- buildMyLocalOrders(&ws, myID)

mainLoop:
	for {
		changed := false

		select {
		case btn := <-buttonCh:
			ownerID := ownerForButton(myID, btn.Button)
			key := makeOrderKey(ownerID, btn.Floor, btn.Button)

			info := localOrderView[key]
			if info.Phase == Unconfirmed || info.Phase == Confirmed {
				continue mainLoop
			}

			info.Phase = Unconfirmed
			info.SeenBy = map[string]bool{myID: true}
			localOrderView[key] = info

			if allAliveHaveSeen(info.SeenBy, ws.Alive) {
				if confirmOrderInWorldState(&ws, key) {
					changed = true
				}
				info.Phase = Confirmed
				info.SeenBy = map[string]bool{myID: true}
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

				if info.Phase != Confirmed {
					continue
				}

				// Sett til Served lokalt
				info.Phase = Served
				info.SeenBy = map[string]bool{myID: true}
				localOrderView[key] = info

				// Clear worldstate lokalt (idempotent)
				if clearOrderInWorldState(&ws, key) {
					changed = true
				}
				changed = true

				if allAliveHaveSeen(info.SeenBy, ws.Alive) {
					delete(localOrderView, key)
				}
				/* if key.Button == config.BT_Cab && key.OwnerID == myID {
					delete(localOrderView, key)
				} */
			}

		case st := <-localStateCh:
			prev := ws.States[myID]
			st.ID = myID
			ws.States[st.ID] = st

			if prev.Immobile != st.Immobile {
				changed = true
			}

		case pst := <-peerStateCh:
			prev := ws.States[pst.ID]
			ws.States[pst.ID] = pst

			if prev.Immobile != pst.Immobile {
				changed = true
			}

		case pe := <-peerEventCh:
			prev, exists := ws.Alive[pe.PeerID]
			if !exists || prev != pe.Alive {
				ws.Alive[pe.PeerID] = pe.Alive
				changed = true
			}
			//fmt.Printf("[OM %s] PEER EVENT peer=%s alive=%v ws.Alive=%+v\n", myID, pe.PeerID, pe.Alive, ws.Alive)

			if pe.Alive {
				for key, info := range localOrderView {
					if key.OwnerID == myID && key.Button == config.BT_Cab && info.Phase == Served {
						delete(localOrderView, key)
					}
				}
			}

		case peerOrder := <-OrderInCh:
			key := makeOrderKey(peerOrder.OwnerID, peerOrder.Floor, peerOrder.Button)
			info := localOrderView[key]
			if info.SeenBy == nil {
				info.SeenBy = make(map[string]bool)
				info.Phase = NoOrder
			}
			//fmt.Printf("[OM %s] BEFORE MERGE key=%+v incomingPhase=%v localPhase=%v incomingSeenBy=%+v ws.Alive=%+v\n", myID, key, peerOrder.Phase, info.Phase, peerOrder.SeenBy, ws.Alive)

			// Vanlig fase-sammenligning
			if peerOrder.Phase > info.Phase {
				info.Phase = peerOrder.Phase
				info.SeenBy = make(map[string]bool)
			} else if peerOrder.Phase < info.Phase {
				//localOrderView[key] = info
				continue mainLoop
			}

			for id, seen := range peerOrder.SeenBy {
				if seen {
					info.SeenBy[id] = true
				}
			}
			info.SeenBy[myID] = true
			localOrderView[key] = info

			if info.Phase == Unconfirmed && !allAliveHaveSeen(info.SeenBy, ws.Alive) {
				continue mainLoop
			}

			switch info.Phase {
			case Unconfirmed:
				if confirmOrderInWorldState(&ws, key) {
					changed = true
				}
				info.Phase = Confirmed
				info.SeenBy = map[string]bool{myID: true}
				localOrderView[key] = info
				changed = true

			case Confirmed:
				// Recovery/rejoin: sørg for at WorldState er i sync
				if confirmOrderInWorldState(&ws, key) {
					changed = true
				}
				//localOrderView[key] = info

			case Served:
				// Idempotent clear på alle noder
				if clearOrderInWorldState(&ws, key) {
					changed = true
				}
				delete(localOrderView, key)
				changed = true
			}

		case <-SendTicker.C:
			for key, info := range localOrderView {
				if info.Phase == NoOrder {
					continue
				}
				//fmt.Printf("[OM %s] TX ORDER key=%+v phase=%v seenBy=%+v\n",myID, key, info.Phase, info.SeenBy)
				OrderOutCh <- OrderMsg{
					OwnerID: key.OwnerID,
					Floor:   key.Floor,
					Button:  key.Button,
					Phase:   info.Phase,
					SeenBy:  copySeenBy(info.SeenBy),
				}
			}
		}

		if changed {
			setButtonLight <- buildLightState(&ws, myID)
			orders := buildMyLocalOrders(&ws, myID)
			//fmt.Printf("[OM %s] ordersOut cab%v hall=%v\n", myID, orders.Cab, orders.Hall)
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

/* func isConfirmedInWorldState(ws *WorldState, key OrderKey) bool {
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
} */

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
	//fmt.Printf("MyAssignedHall: %v\n", myAssignedHall)
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
