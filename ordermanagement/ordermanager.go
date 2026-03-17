package ordermanagement

import (
	"heis/config"
	"time"
)

func Run(
	myID string,
	buttonPressedCh <-chan config.ButtonEvent,
	clearCh <-chan config.ClearEvent,
	localStateCh <-chan config.ElevatorState,
	peerStateCh <-chan config.ElevatorState,
	peerAlivenessCh <-chan config.PeerEvent,
	ordersToFsmCh chan<- Orders,
	OrdersBroadcastCh chan<- OrderMsg, // OM -> Network
	OrdersFromNetwork <-chan OrderMsg, // OM <- Network
	setButtonLight chan<- config.LightState,
) {


	//init?
	worldState := NewWorldState()
	localOrderView := make(OrderRegister)

	initLocalElevator(&worldState, myID)

	initialOrders := buildMyLocalOrders(&worldState, myID)
	ordersToFsmCh <- initialOrders

	orderBroadcastTicker := time.NewTicker(100 * time.Millisecond)
	defer orderBroadcastTicker.Stop()
	

mainLoop:
	for {
		changed := false

		select {
		case btn := <-buttonPressedCh:

			//lage en funksjon som gjør dette?
			ownerID := ownerForButton(myID, btn.Button)
			key := makeOrderKey(ownerID, btn.Floor, btn.Button)
			localOrder := localOrderView[key]

			if localOrder.Phase == Unconfirmed || localOrder.Phase == Confirmed {
				continue mainLoop
			}

			localOrder = setLocalOrderPhase(localOrderView, key, Unconfirmed, myID)

			if allAliveHaveSeen(localOrder.SeenBy, worldState.Alive) {
				if confirmOrderInWorldState(&worldState, key) {
					changed = true
				}
				
				localOrder = setLocalOrderPhase(localOrderView, key, Confirmed, myID)
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

				//lage en funksjon som gjør dette?
				ownerID := ownerForButton(myID, clearInfo.button)
				key := makeOrderKey(ownerID, cl.Floor, clearInfo.button)
				localOrder := localOrderView[key]
			

				if localOrder.Phase != Confirmed {
					continue
				}

				localOrder = setLocalOrderPhase(localOrderView, key, Served, myID)

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

		case peer := <-peerAlivenessCh:
					//her vil jeg kalle kanalen peerAlivenesschange, inni peerevent så skal jeg fjerne peer før id, og døpe om peerevent til peeralivness
			previousLiveness := worldState.Alive[peer.PeerID]
			incomingLiveness := peer.Alive

			if previousLiveness != incomingLiveness {
				worldState.Alive[peer.PeerID] = incomingLiveness
				changed = true
			}
			
			//denne må fikses!! chat melder at den er smart, men den er litt random også- så vabnskelig å navngi
			if peer.Alive {
				for key, info := range localOrderView {
					if key.OwnerID == myID && key.Button == config.BT_Cab && info.Phase == Served {
						delete(localOrderView, key)
					}
				}
			}



		case peerOrder := <-OrdersFromNetwork:
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
				localOrder = setLocalOrderPhase(localOrderView, key, Confirmed, myID)
				changed = true


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



		case <-orderBroadcastTicker.C: 
			for key, localOrder := range localOrderView {
				if localOrder.Phase == NoOrder {
					continue
				}
				OrdersBroadcastCh <- OrderMsg{
					OwnerID: key.OwnerID,
					Floor:   key.Floor,
					Button:  key.Button,
					Phase:   localOrder.Phase,
					SeenBy:  copySeenByMap(localOrder.SeenBy),
				}
			}
		}

		if changed {
			setButtonLight <- buildLightState(&worldState, myID)
			ordersToFsmCh <- buildMyLocalOrders(&worldState, myID)
		}
	}
}

//-------


func initLocalElevator(ws *WorldState, myID string) {
	ws.Alive[myID] = true

	ws.States[myID] = config.ElevatorState{
		ID:          myID,
		Floor:       0,
		Behaviour:   config.BehIdle,
		Direction:   config.DirStop,
		CabRequests: make([]bool, config.N_FLOORS),
	}
}
